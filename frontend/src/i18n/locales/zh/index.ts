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
    home: { title: 'AI 模型 API 网关', description: '通过统一 API 接入 AI 模型，查看模型定价和接入指南。' },
    market: { title: '模型市场', description: '按分组比较 AI 模型的输入、输出和缓存价格。' },
    docs: { title: 'API 文档', description: '查看 AI API 接入、客户端配置和 API 密钥管理指南。' },
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
