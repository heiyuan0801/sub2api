import design from './design'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import publicSite from './publicSite'
import publicHome from './publicHome'

import cluster from './cluster'

export default {
  ...design,
  ...cluster,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
  ...publicSite,
  ...publicHome,
}
