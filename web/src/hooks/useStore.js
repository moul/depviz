import React, {
  createContext, useContext, useMemo, useState,
} from 'react'

const DEFAULT_STATE = {
  apiData: null,
  layout: null,
}

function createContextValue(state, setState) {
  return {
    ...state,
    updateApiData: (data, layout) => {
      setState({ ...state, apiData: data, layout })
    },
    updateLayout: (layout) => {
      setState({ ...state, layout })
    },
  }
}

const StoreContext = createContext(createContextValue({
  ...DEFAULT_STATE,
  setState: () => console.error('You are using StoreContext without StoreProvider!'),
}))

export function useStore() {
  return useContext(StoreContext)
}

export function StoreProvider({ context, children }) {
  // console.log('authContext: ', context)
  const [state, setState] = useState({
    ...DEFAULT_STATE,
    ...context,
  })

  // Memoize context values
  const contextValue = useMemo(() => createContextValue(state, setState), [state, setState])

  return (<StoreContext.Provider value={contextValue}>{children}</StoreContext.Provider>)
}
