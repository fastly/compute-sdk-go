// Node & 3P Modules
const fs = require('fs');
const path = require('path');
const childProcess = require('child_process');
const { Worker } = require("worker_threads");
const Comlink = require('comlink');
const nodeAdapter = require('comlink/dist/umd/node-adapter.js');
const fetch = require('node-fetch');
const pWaitFor = require('p-wait-for');
const pLimit = require('p-limit').default;
const chalk = require('chalk');
const logSymbols = require('log-symbols').default;
require('better-logging')(console, {
  format: ctx => {

    const tag = chalk.bold(`[compute-sdk-test]`); 

    if (ctx.type.includes("debug")) {
      return chalk.blue(`${tag} ${chalk.bold(ctx.type)} ${ctx.msg}`);
    } else if (ctx.type.includes("info")) {
      return chalk.green(`${logSymbols.info} ${tag} ${chalk.bold(ctx.type)} ${ctx.msg}`);
    } else if (ctx.type.includes("warn")) {
      return chalk.yellow(`${logSymbols.warning} ${tag} ${chalk.bold(ctx.type)} ${ctx.msg}`);
    } else if (ctx.type.includes("error")) {
      return chalk.red(`${logSymbols.error} ${tag} ${chalk.bold(ctx.type)} ${ctx.msg}`);
    }

    return `${tag} ${chalk.white.bold(ctx.type)} ${ctx.msg}`;
  }
});
const core = require('@actions/core');
const github = require('@actions/github');

// Utility modules
const Viceroy = require('./src/viceroy.js');
const LiveComputeTest = require('./src/live-compute-test.js');
const UpstreamServer = require('./src/upstream-server.js');
const compareUpstreamRequest = require('./src/compare-upstream-request.js');
const compareDownstreamResponse = require('./src/compare-downstream-response.js');

// Get our config from the Github Action
const configRelativePath = `${core.getInput("config")}`;
console.info(`Parsing SDK Test config: ${configRelativePath}`);
const configAbsolutePath = path.resolve(configRelativePath);
const config = JSON.parse(fs.readFileSync(configAbsolutePath));
console.info('Running the SDK Config:');
console.log(`${JSON.stringify(config, null, 2)}`);

// Get our token from the Github Action
const fastlyToken = `${core.getInput("fastly_token")}`;

// Our main task, in which we compile and run tests
const mainAsyncTask = async () => {
  // Iterate through our config and compile our wasm modules
  const modules = config.modules;
  const moduleKeys = Object.keys(modules);

  moduleKeys.forEach(key => {
    const module = modules[key];
    console.info(`Compiling the fixture for: ${key} ...`);
    const moduleBuildStdout = childProcess.execSync(
      module.build,
      {
        stdio: 'inherit'
      }
    );
  });

  console.info(`Running the Viceroy environment tests ...`);

  // Define our viceroy here so we can kill it on errors
  let viceroy;

  // Check if we are validating any local upstream requests (For example, like telemetry being sent)
  // If so, we will need an upstream server to compare the request that was sent
  let isDownstreamResponseHandled = false;
  let upstreamServer = new UpstreamServer();
  let upstreamServerTest;
  upstreamServer.listen(8081, async (localUpstreamRequestNumber, req, res) => {
    // Let's do the verifications on the request
    const configRequest = upstreamServerTest.local_upstream_requests[localUpstreamRequestNumber];

    try {
      await compareUpstreamRequest(configRequest, req, isDownstreamResponseHandled);
    } catch (err) {
      await viceroy.kill();
      console.error(`[LocalUpstreamRequest (${localUpstreamRequestNumber})] ${err.message}`);
      process.exit(1);
    }
  });

  // Iterate through the module tests, and run the Viceroy tests
  for (const moduleKey of moduleKeys) {
    const module = modules[moduleKey];
    const moduleTestKeys = Object.keys(module.tests);
    console.info(`Running tests for the module: ${moduleKey} ...`);

    // Spawn a new viceroy instance for the module
    viceroy = new Viceroy();
    const viceroyAddr = '127.0.0.1:8080';
    await viceroy.spawn(module.wasm_path, {
      config: module.fastly_toml_path,
      addr: viceroyAddr
    })


    for (const testKey of moduleTestKeys) {
      const test = module.tests[testKey];

      // Check if this module should be tested in viceroy
      if (!test.environments.includes("viceroy")) {
        continue;
      }

      console.log(`Running the test ${testKey} ...`);

      // Prepare our upstream server for this specific test
      isDownstreamResponseHandled = false;
      if (test.local_upstream_requests) {
        upstreamServerTest = test;
        upstreamServer.setExpectedNumberOfRequests(test.local_upstream_requests.length);
      } else {
        upstreamServerTest = null;
        upstreamServer.setExpectedNumberOfRequests(0);
      }

      // Make the downstream request to the server (Viceroy)
      const downstreamRequest = test.downstream_request;
      let downstreamResponse;
      try {
        downstreamResponse = await fetch(`http://${viceroyAddr}${downstreamRequest.pathname || ''}`, {
          method: downstreamRequest.method || 'GET',
          headers: downstreamRequest.headers || undefined,
          body: downstreamRequest.body || undefined
        });
      } catch(error) {
        await upstreamServer.close();
        await viceroy.kill();
        console.error(error);
        process.exit(1);
      }

      // Now that we have gotten our downstream response, we can flip our boolean
      // that our local_upstream_request will check
      isDownstreamResponseHandled = true;

      // Check the Logs to see if they match expected logs in the config
      if (test.logs) {
        for (let i = 0; i < test.logs.length; i++) {
          let log = test.logs[i];

          if (!viceroy.logs.includes(log)) {
            console.error(`[Logs: log not found] Expected: ${log}`);
            await upstreamServer.close();
            await viceroy.kill();
            process.exit(1);
          }
        }
      }

      // Do our confirmations about the downstream response
      const configResponse = test.downstream_response;
      try {
        await compareDownstreamResponse(configResponse, downstreamResponse);
      } catch (err) {
        console.error(err.message);
        await upstreamServer.close();
        await viceroy.kill();
        process.exit(1);
      }

      console.log(`The test ${testKey} Passed!`);

      // Done! Kill the process, and go to the next test
      try {
        await upstreamServer.waitForExpectedNumberOfRequests();
        upstreamServerTest = null;
        upstreamServer.setExpectedNumberOfRequests(0);
      } catch(e) {
        console.error('Could not cleanup the upstream server. Error Below:');
        console.error(e);
        process.exit(1);
      }
    }

    // Kill Viceroy and continue onto the next module
    try {
      await viceroy.kill();
    } catch(e) {
      console.error('Could not kill Viceory. Error Below:');
      console.error(e);
      process.exit(1);
    }
  };

  // Viceroy is done! Close our upstream server and things
  await upstreamServer.close();

  // Check if we have C@E Environement tests
  let shouldRunComputeTests = moduleKeys.some(moduleKey => {
    const module = modules[moduleKey];
    const moduleTestKeys = Object.keys(module.tests);

    return moduleTestKeys.some(testKey => {

      const test = module.tests[testKey];
      // Check if this module should be tested in viceroy
      if (test.environments.includes("c@e")) {
        return true;
      }
      return false;
    });
  });

  if (!shouldRunComputeTests) {
    console.info('Viceroy environment tests are done, and no C@E environment tests!');
    console.info('We are finished, all tests passed! :)');
    return;
  }

  console.info('Viceroy environment tests are done! Now doing C@E Tests...');

  if (!fastlyToken) {
    console.warn("Fastly Token was not found in the C@E SDK CI Github Action");
    console.warn("Please pass a Fastly Token to test on the C@E enviornment...")
    return;
  }

  await LiveComputeTest.buildWaitingApp();
  console.log('Installing dependencies for our Workers...');
  childProcess.execSync(
    `(cd ${__dirname}/.. && npm install)`,
    {
      stdio: 'inherit'
    }
  );
  const modulePromiseLimit = pLimit(config.services.length);
  const moduleTestPromises = [];

  // Iterate through the modules we are testing in the config
  // And then test them on C@E
  for (const moduleKey of moduleKeys) {
    const module = modules[moduleKey];
    const moduleTestKeys = Object.keys(module.tests);

    let hasComputeTest = moduleTestKeys.some(testKey => {
      const test = module.tests[testKey];
      // Check if this module should be tested in viceroy
      if (test.environments.includes("c@e")) {
        return true;
      }
      return false;
    });

    if (!hasComputeTest) {
      continue;
    }


    // https://www.npmjs.com/package/comlink
    // https://alvinlal.netlify.app/blog/single-thread-vs-child-process-vs-worker-threads-vs-cluster-in-nodejs
    moduleTestPromises.push(modulePromiseLimit(async () => {

      // Find an unused service
      let service = null;
      config.services.some(configService => {
        if (!configService.isActivelyUsed) {
          service = configService;
          service.isActivelyUsed = true;
          return true;
        }
        return false;
      });

      if (!service) {
        throw new Error(`Could not run the tests for ${moduleKey} , all services being used ...`)
      }

      console.info(`Running the C@E test for ${moduleKey} , on the service ${service.id} ...`);

      // We need to parallelize the actual Live Compute Test, as deploying the serivces is quite slow.
      // So we are using a combination of Web Workers and comlink, but child_process.fork could also work.
      // https://www.npmjs.com/package/comlink
      // https://alvinlal.netlify.app/blog/single-thread-vs-child-process-vs-worker-threads-vs-cluster-in-nodejs
      try {
        const worker = new Worker(`${__dirname}/../compute-test-worker.js`)
        const computeTestWorker = Comlink.wrap(nodeAdapter(worker));
        await computeTestWorker.run(moduleKey, module, fastlyToken, service);
        await worker.terminate();
      } catch(e) {
        console.error(e);
        throw new Error(e);
      }

      service.isActivelyUsed = false;
    }));
    // Done!
  };

  // Run the actual promises
  try {
    await Promise.all(moduleTestPromises);

    // We are done! Exit!
    process.exit(0);
  } catch(e) {
    console.error(e);
    process.exit(1);
  }
};
mainAsyncTask().then(() => {
  process.exit(0);
}).catch((error) => {
  core.setFailed(error.message)
  process.exit(1);
});


