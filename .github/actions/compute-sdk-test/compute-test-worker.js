const { parentPort } = require('worker_threads');
const Comlink = require('comlink');
const nodeEndpoint = require('comlink/dist/umd/node-adapter.js');
const fetch = require('node-fetch');

const LiveComputeTest = require('./src/live-compute-test.js');
const compareDownstreamResponse = require('./src/compare-downstream-response.js');

async function runModuleTest(testKey, test, service, liveComputeTest) {
  console.log(`Running the test: ${testKey} ...`);

  // Make the downstream request to the live C@E App
  const downstreamRequest = test.downstream_request;
  let downstreamResponse;
  try {
    const downstreamUrl = `https://${service.domain}${downstreamRequest.pathname || ''}`;
    downstreamResponse = await fetch(downstreamUrl, {
      method: downstreamRequest.method || 'GET',
      headers: downstreamRequest.headers || undefined,
      body: downstreamRequest.body || undefined
    });
  } catch(error) {
    console.error(error);
    // Deploy the waiting app for cleanup
    await liveComputeTest.deployWaitingApp();
    throw new Error(error.message);
  }

  // Do our confirmations about the downstream response
  const configResponse = test.downstream_response;
  try {
    await compareDownstreamResponse(configResponse, downstreamResponse);
  } catch (error) {
    console.error(error.message);
    // Deploy the waiting app for cleanup
    await liveComputeTest.deployWaitingApp();
    throw new Error(error.message);
  }

  console.log(`Test: "${testKey}" Passed!`);
}

const ComputeTestWorker = {
  run: async (moduleKey, module, fastlyToken, service) => {
    let liveComputeTest = new LiveComputeTest(module, fastlyToken, service);

    // Check that the service is ready and waiting for a deployment
    try {
      let downstreamResponse = await fetch(`https://${service.domain}/`);
      if (!downstreamResponse.headers.has('compute-sdk-test-waiting')) {
        throw new Error('Waiting app is not live...');
      }
    } catch(error) {
      if (error.message.includes('Waiting app')) { 
        // The error was that the waiting app wasn't live, so some cleanup was missed,
        // so let's deploy the waiting app, and then continue the test as normal
        await liveComputeTest.deployWaitingApp();
      } else {
        console.error(error);
        throw new Error(error.message);
      }
    }

    console.log('Deploying the module....');

    // Deploy the wasm module to C@E
    await liveComputeTest.deployTestApp();

    // Run the test on the module
    console.info(`Testing the module: "${moduleKey}" ...`);

    const moduleTestKeys = Object.keys(module.tests);
    for (const testKey of moduleTestKeys) {
      const test = module.tests[testKey];

      // Check if this module should be tested in C@E
      if (!test.environments.includes("c@e")) {
        continue;
      }

      await runModuleTest(testKey, test, service, liveComputeTest);
    }

    console.log('All module tests passed!');

    // Deploy the waiting app for cleanup
    await liveComputeTest.deployWaitingApp();
  }
};

Comlink.expose(ComputeTestWorker, nodeEndpoint(parentPort));
