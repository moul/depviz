const storageFactory = (storage) => {
  let inMemoryStorage = {}
  const length = 0

  function isSupported() {
    try {
      const testKey = '__some_random_key_you_are_not_going_to_use__'
      storage.setItem(testKey, testKey)
      storage.removeItem(testKey)
      return true
    } catch (e) {
      return false
    }
  }

  function clear() {
    if (isSupported()) {
      storage.clear()
    } else {
      inMemoryStorage = {}
    }
  }

  function getItem(name) {
    if (isSupported()) {
      return storage.getItem(name)
    }

    if (Object.prototype.hasOwnProperty.call(inMemoryStorage, name)) {
      return inMemoryStorage[name]
    }
    return null
  }

  function key(index) {
    if (isSupported()) {
      return storage.key(index)
    }
    return Object.keys(inMemoryStorage)[index] || null
  }

  function removeItem(name) {
    if (isSupported()) {
      storage.removeItem(name)
    } else {
      delete inMemoryStorage[name]
    }
  }

  function setItem(name, value) {
    if (isSupported()) {
      storage.setItem(name, value)
    } else {
      inMemoryStorage[name] = String(value) // not everyone uses TypeScript
    }
  }

  return {
    getItem,
    setItem,
    removeItem,
    clear,
    key,
    length,
  }
}

const store = storageFactory(typeof window !== 'undefined' ? localStorage : null)

export default store
