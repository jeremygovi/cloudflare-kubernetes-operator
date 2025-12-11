import { KubeConfig, CustomObjectsApi, Watch } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from './utils/logger';
import { CloudflareRecordController } from './controllers/cloudflare-record-controller';
import { CloudflareRulesetController } from './controllers/cloudflare-ruleset-controller';

export class CloudflareOperator {
  private kubeConfig: KubeConfig;
  private k8sApi: CustomObjectsApi;
  private cloudflare: Cloudflare;
  private recordController: CloudflareRecordController;
  private rulesetController: CloudflareRulesetController;

  constructor() {
    this.kubeConfig = new KubeConfig();

    // Load kubeconfig (in-cluster or from file)
    if (process.env.KUBERNETES_SERVICE_HOST) {
      logger.info('Running in-cluster, loading in-cluster config');
      this.kubeConfig.loadFromCluster();
    } else {
      logger.info('Running outside cluster, loading from default kubeconfig');
      this.kubeConfig.loadFromDefault();
    }

    this.k8sApi = this.kubeConfig.makeApiClient(CustomObjectsApi);

    // Initialize Cloudflare client
    const apiToken = process.env.CLOUDFLARE_API_TOKEN;
    if (!apiToken) {
      throw new Error('CLOUDFLARE_API_TOKEN environment variable is required');
    }

    const accountId = process.env.CLOUDFLARE_ACCOUNT_ID;
    if (accountId) {
      logger.info(`Using Cloudflare Account ID: ${accountId.substring(0, 8)}...`);
    }

    this.cloudflare = new Cloudflare({ apiToken });

    // Initialize controllers
    this.recordController = new CloudflareRecordController(
      this.k8sApi,
      this.cloudflare
    );
    this.rulesetController = new CloudflareRulesetController(
      this.k8sApi,
      this.cloudflare
    );
  }

  async start() {
    logger.info('Starting Cloudflare Kubernetes Operator...');
    
    try {
      // Start watching CloudflareRecord resources
      await this.recordController.watch();
      logger.info('Watching CloudflareRecord resources');

      // Start watching CloudflareRuleset resources
      await this.rulesetController.watch();
      logger.info('Watching CloudflareRuleset resources');

      logger.info('Cloudflare Kubernetes Operator is running');
    } catch (error) {
      logger.error('Failed to start operator:', error);
      throw error;
    }
  }

  async stop() {
    logger.info('Stopping Cloudflare Kubernetes Operator...');
    this.recordController.stop();
    this.rulesetController.stop();
    logger.info('Operator stopped');
  }
}

// Handle graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', async () => {
  logger.info('SIGINT received, shutting down gracefully');
  process.exit(0);
});
