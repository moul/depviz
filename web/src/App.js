/* eslint-disable import/no-named-as-default */
import { Route, Switch, BrowserRouter as Router } from 'react-router-dom'
import React, { useEffect } from 'react'
import { hot } from 'react-hot-loader'
import Stats from 'stats.js'
import { StoreProvider, useStore } from './hooks/useStore'
import HomePage from './ui/pages/HomePage/HomePage'
import Menu from './ui/components/Header/Menu'

// Import Tabler styles
import './assets/scss/tabler.scss'

import './App.scss'

const showDebug = process.env.NODE_ENV === 'development'

const App = () => {
  const {
    debugInfo,
  } = useStore()

  useEffect(() => {
    if (showDebug) {
      const stats = new Stats()
      stats.showPanel(0) // 0: fps, 1: ms, 2: mb, 3+: custom
      stats.dom.id = 'debug-info'
      document.body.appendChild(stats.dom)

      const animate = () => {
        stats.begin()
        // monitored code goes here

        stats.end()

        requestAnimationFrame(animate)
      }

      requestAnimationFrame(animate)
    }
  })

  return (
    <StoreProvider>
      <div className="page">
        <div className="flex-fill">
          <Menu />
          <Router>
            <Switch>
              <Route exact path="/" component={HomePage} />
            </Switch>
          </Router>
        </div>
        {/* <footer className="footer">
        <div className="container">
          <div className="row align-items-center flex-row-reverse">
            <div className="col-12 col-lg-auto mt-3 mt-lg-0 text-center">
              {' '}
            </div>
          </div>
        </div>
      </footer> */}
        {showDebug && (
        <div className="debug-info">
          <div>
            nodes:
            {' '}
            {debugInfo.nodes}
          </div>
          <div>
            edges:
            {' '}
            {debugInfo.edges}
          </div>
        </div>
        )}
      </div>
    </StoreProvider>
  )
}

export default hot(module)(App)
