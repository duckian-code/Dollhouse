import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';
import type { KeyValueStorageInterface } from 'aws-amplify/utils';

const KEY_INDEX = 'dollhouse.auth.keys';
const memoryStorage = new Map<string, string>();
const webKeys = new Set<string>();

function getWebStorage() {
  return typeof localStorage === 'undefined' ? undefined : localStorage;
}

async function getStoredKeys() {
  const value = await SecureStore.getItemAsync(KEY_INDEX);
  return value ? (JSON.parse(value) as string[]) : [];
}

async function rememberKey(key: string) {
  const keys = await getStoredKeys();
  if (!keys.includes(key)) {
    await SecureStore.setItemAsync(KEY_INDEX, JSON.stringify([...keys, key]));
  }
}

export const authStorage: KeyValueStorageInterface = {
  async setItem(key, value) {
    if (Platform.OS === 'web') {
      const storage = getWebStorage();
      if (storage) storage.setItem(key, value);
      else memoryStorage.set(key, value);
      webKeys.add(key);
      return;
    }
    await SecureStore.setItemAsync(key, value);
    await rememberKey(key);
  },
  async getItem(key) {
    if (Platform.OS === 'web') {
      return getWebStorage()?.getItem(key) ?? memoryStorage.get(key) ?? null;
    }
    return SecureStore.getItemAsync(key);
  },
  async removeItem(key) {
    if (Platform.OS === 'web') {
      getWebStorage()?.removeItem(key);
      memoryStorage.delete(key);
      webKeys.delete(key);
      return;
    }
    await SecureStore.deleteItemAsync(key);
    const keys = await getStoredKeys();
    await SecureStore.setItemAsync(
      KEY_INDEX,
      JSON.stringify(keys.filter((storedKey) => storedKey !== key)),
    );
  },
  async clear() {
    if (Platform.OS === 'web') {
      const storage = getWebStorage();
      webKeys.forEach((key) => storage?.removeItem(key));
      webKeys.clear();
      memoryStorage.clear();
      return;
    }
    const keys = await getStoredKeys();
    await Promise.all(keys.map((key) => SecureStore.deleteItemAsync(key)));
    await SecureStore.deleteItemAsync(KEY_INDEX);
  },
};
