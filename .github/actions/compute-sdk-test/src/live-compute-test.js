const pWaitFor = require('p-wait-for');
const fetch = require('node-fetch');

const childProcess = require('child_process');

// LiveComputeTest - 
class LiveComputeTest {

  static SECONDS = 1000;
  static MINUTES = 60 * LiveComputeTest.SECONDS;

  static buildWaitingApp() {
    // Build the waiting app
    console.log('Building the waiting app...');
    childProcess.execSync(
      `(cd ${__dirname}/../../compute-sdk-test-waiting && npm install && npm run build && npm run pack)`,
      {
        stdio: 'inherit'
      }
    );
  }

  constructor(module, fastlyToken, service) {
    this.module = module;
    this.fastlyToken = fastlyToken;
    this.service = service;
  }

  async deployWaitingApp() {
    // We need to deploy the waiting app, and then wait for it, before we continue
    console.log('Deploying the waiting app....');
    childProcess.execSync(
      `pwd && fastly compute deploy --token ${this.fastlyToken} --service-id ${this.service.id} --package ${__dirname}/../../compute-sdk-test-waiting/pkg/compute-sdk-test-waiting.tar.gz`,
      {
        stdio: 'inherit'
      }
    );

    // Poll the app until the waiting app is live
    console.log('Polling until the waiting app is live...');
    await pWaitFor(async () => {
      let downstreamResponse = await fetch(`https://${this.service.domain}/`);
      return downstreamResponse.headers.has('compute-sdk-test-waiting');
    }, {
      interval: 5 * LiveComputeTest.SECONDS, 
      timeout: 5 * LiveComputeTest.MINUTES
    });
  }

  async deployTestApp() {
    // Deploy the wasm module to C@E
    childProcess.execSync(
      `fastly compute deploy --token ${this.fastlyToken} --service-id ${this.service.id} --package ${this.module.pkg_path}`,
      {
        stdio: 'inherit'
      }
    );

    // Poll the app until the waiting app is gone
    console.log('Polling until the test is live...');
    await pWaitFor(async () => {
      let downstreamResponse = await fetch(`https://${this.service.domain}/`);
      return !downstreamResponse.headers.has('compute-sdk-test-waiting');
    }, {
      interval: 5 * LiveComputeTest.SECONDS, 
      timeout: 5 * LiveComputeTest.MINUTES
    });
    // Also, there is a period of time while the new app is deploying, that you will get 500s, wait until that stops,
    // Or, it will timeout and actually run the 500
    try {
      await pWaitFor(async () => {
        let downstreamResponse = await fetch(`https://${this.service.domain}/`);
        return downstreamResponse.status === 200
      }, {
        interval: 5 * LiveComputeTest.SECONDS, 
        timeout: 30 * LiveComputeTest.SECONDS
      });
    } catch(e) {
      // Continue with the 500 because it might be what we are testing
    }
  }
}


module.exports = LiveComputeTest; 
