import { Platform } from 'react-native';

import { nativeCacheDatabase } from '@/services/cache/database.native';
import { webCacheDatabase } from '@/services/cache/database.web';

export const cacheDatabase =
  Platform.OS === 'web' ? webCacheDatabase : nativeCacheDatabase;
