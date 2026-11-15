import design from './design'
import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import publicSite from './publicSite'
import publicHome from './publicHome'
import rewards from './rewards'
import docs from './docs'
import operations from './operations'
import planOps from './planOps'

import cluster from './cluster'

export default {
  ...planOps,
  seo: {
    home: { title: 'AI Model API Gateway', description: 'Access AI models through a unified API and browse model pricing and integration guides.' },
    market: { title: 'Model Market', description: 'Compare AI models and see input, output and cache prices by group.' },
    docs: { title: 'API Documentation', description: 'Learn how to connect to the AI API, configure clients and manage API keys.' },
  },
  ...design,
  ...cluster,
  adminAccessKeys,
  mcp,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
  ...publicSite,
  ...publicHome,
  rewards,
  docs,
  operations,
}
