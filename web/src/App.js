/* eslint-disable import/no-named-as-default */
import { Route, Switch, BrowserRouter as Router } from 'react-router-dom'
import React, { useEffect, useState } from 'react'
import { hot } from 'react-hot-loader'
import Stats from 'stats.js'
import { StoreProvider, useStore } from './hooks/useStore'
import HomePage from './ui/pages/HomePage/HomePage'
import Menu from './ui/components/Header/Menu'
import Modal from './ui/components/Modal/Modal'
import store from './utils/store'

// Import Tabler styles
import './assets/scss/tabler.scss'

import './App.scss'

const showDebug = process.env.NODE_ENV === 'development'

const App = () => {
  const {
    debugInfo,
  } = useStore()

  const [showAuthModal, setShowAuthModal] = useState(!store.getItem('auth_token'))
  const [authToken, setAuthToken] = useState(store.getItem('auth_token') || '')

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

  const handleChange = (e) => {
    e.preventDefault()
    const token = event.target.value
    store.setItem('auth_token', token)
    setAuthToken(token)
    // setShowAuthModal(!token)
  }

  const handleClose = (e) => {
    e.preventDefault()
    setShowAuthModal(false)
  }
  return (
    <StoreProvider>
      <div className="page">
        <div className="flex-fill">
          <Menu authToken={authToken} handleShowToken={() => setShowAuthModal(true)} />
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
        <Modal
          showModal={showAuthModal}
          id="auth-modal"
          size="lg"
        >
          <div className="modal-header">
            <h5 className="modal-title">Enter auth token</h5>
            <button type="button" className="close" data-dismiss="modal" aria-label="Close" onClick={handleClose}>
              <i className="fe fe-x" />
              x
            </button>
          </div>
          <div className="modal-body">
            <p>Enter your auth token below.</p>
            <form onSubmit={handleClose}>
              <input type="text" name="authToken" id="authToken" placeholder="Auth token" className="form-control" value={authToken} onChange={handleChange} />
              <br />
              <button type="submit" className="btn btn-primary" data-dismiss="modal">Save auth token</button>
            </form>
          </div>
        </Modal>
      </div>
    </StoreProvider>
  )
}

export default hot(module)(App)
