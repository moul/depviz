import React, {
  createContext, useContext, useMemo, useState,
} from 'react'

const DEFAULT_STATE = {
  apiData: {},
  layout: {
    name: 'circle',
    avoidOverlap: true,
  },
  repName: 'moul-bot/depviz-test',
}

function createContextValue(state, setState) {
  let layoutConfig = {}
  const computeLayoutConfig = (layout) => {
    switch (layout) {
      case 'circle':
        layoutConfig = {
          name: 'circle',
          avoidOverlap: true,
        }
        break
      case 'cose':
        layoutConfig = {
          name: 'cose',
          animate: false,
          componentSpacing: 0.5,
          nodeOverlap: 2,
          nodeRepulsion: 0.5,
          nestingFactor: 19,
          gravity: 200,
          numIter: 2000,
          coolingFactor: 0.2,
        }
        break
      case 'breadthfirst':
        layoutConfig = {
          name: 'breadthfirst',
        }
        break
      case 'concentric':
        layoutConfig = {
          name: 'concentric',
        }
        break
      case 'grid':
        layoutConfig = {
          name: 'grid',
          condense: true,
        }
        break
      case 'random':
        layoutConfig = {
          name: 'random',
        }
        break
      case 'cola':
        layoutConfig = {
          name: 'cola',
          animate: false,
          refresh: 1,
          padding: 30,
          maxSimulationTime: 100,
        }
        break
      case 'elk':
        layoutConfig = {
          name: 'elk',
          elk: {
            zoomToFit: true,
            algorithm: 'mrtree',
            separateConnectedComponents: false,
          },
        }
        break
      case 'gantt':
        layoutConfig = {
          name: 'gantt',
        }
        break
      case 'flow':
        layoutConfig = {
          name: 'flow',
        }
        break
      default:
        break
    }

    return layoutConfig
  }
  return {
    ...state,
    updateApiData: (data, layout, repName) => {
      setState({
        ...state, apiData: data, layout: computeLayoutConfig(layout), repName,
      })
    },
    updateLayout: (layout) => {
      setState({ ...state, layout: computeLayoutConfig(layout) })
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
