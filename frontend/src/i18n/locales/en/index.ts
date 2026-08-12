import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import custom from './custom'
import { mergeLocaleMessages } from '../merge'

const upstreamMessages = {
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
}

export default mergeLocaleMessages(upstreamMessages, custom)
