import { CloudflareOperator } from './operator';
import { logger } from './utils/logger';

const COMMANDS = {
  start: 'Start the Cloudflare Kubernetes Operator',
  help: 'Display this help message',
  version: 'Display version information'
};

function displayHelp() {
  console.log('\n📦 Cloudflare Kubernetes Operator\n');
  console.log('Manage Cloudflare resources directly from Kubernetes\n');
  console.log('Available commands:\n');
  
  Object.entries(COMMANDS).forEach(([cmd, desc]) => {
    console.log(`  ${cmd.padEnd(15)} ${desc}`);
  });
  
  console.log('\nEnvironment variables:');
  console.log('  CLOUDFLARE_API_TOKEN    Required: Cloudflare API token');
  console.log('  LOG_LEVEL              Optional: Log level (debug, info, warn, error)');
  console.log('\nExamples:');
  console.log('  npm start');
  console.log('  node dist/index.js start');
  console.log('  node dist/index.js help\n');
}

function displayVersion() {
  const pkg = require('../package.json');
  console.log(`\nCloudflare Kubernetes Operator v${pkg.version}\n`);
}

async function main() {
  const command = process.argv[2] || 'help';

  switch (command) {
    case 'start':
      try {
        const operator = new CloudflareOperator();
        await operator.start();

        // Keep the process running
        process.on('SIGTERM', async () => {
          await operator.stop();
          process.exit(0);
        });

        process.on('SIGINT', async () => {
          await operator.stop();
          process.exit(0);
        });
      } catch (error) {
        logger.error('Failed to start operator:', error);
        process.exit(1);
      }
      break;

    case 'version':
      displayVersion();
      break;

    case 'help':
    default:
      displayHelp();
      break;
  }
}

main().catch((error) => {
  logger.error('Unhandled error:', error);
  process.exit(1);
});
