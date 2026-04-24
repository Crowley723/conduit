let hasSessionStorageSupport: boolean | null = null;
const testKey = 'authelia.test';
const testValue = 'foo';

export function sessionStorageAvailable() {
  if (hasSessionStorageSupport !== null) return hasSessionStorageSupport;

  hasSessionStorageSupport = false;

  if (typeof globalThis !== 'undefined' && globalThis.sessionStorage !== null) {
    hasSessionStorageSupport = true;

    try {
      globalThis.sessionStorage.setItem(testKey, testValue);
      globalThis.sessionStorage.removeItem(testKey);
    } catch {
      hasSessionStorageSupport = false;
    }
  }

  return hasSessionStorageSupport;
}

export function getSessionStorage(key: string) {
  if (!sessionStorageAvailable()) return null;

  return globalThis.sessionStorage.getItem(key);
}

export function setSessionStorage(key: string, value: string) {
  if (!sessionStorageAvailable()) return false;

  globalThis.sessionStorage.setItem(key, value);

  return true;
}

export function removeSessionStorage(key: string) {
  if (!sessionStorageAvailable()) return false;

  globalThis.sessionStorage.removeItem(key);

  return true;
}
