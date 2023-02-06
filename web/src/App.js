/* eslint-disable import/no-named-as-default */
import { Route, Switch, BrowserRouter as Router } from 'react-router-dom'
import React, { useState } from 'react'
import { hot } from 'react-hot-loader'
import { XCircle } from 'react-feather'
import { StoreProvider } from './hooks/useStore'
import HomePage from './ui/pages/HomePage/HomePage'
import GitHubOAuthPage from './ui/pages/GitHubOAuthPage/GitHubOAuthPage'
import Menu from './ui/components/Header/Menu'
import Modal from './ui/components/Modal/Modal'
import store from './utils/store'
import computeLayoutConfig from './utils/computeLayoutConfig'

// Import Tabler styles
import 'tabler/scss/tabler.scss'

import './App.scss'

const defaultTargets = process.env.DEFAULT_TARGETS

const App = () => {
  const [showAuthModal, setShowAuthModal] = useState(false) // !store.getItem('auth_token'))
  const [authToken, setAuthToken] = useState(store.getItem('auth_token') || '')
  const [trelloApiKey, setTrelloApiKey] = useState(store.getItem('trello_api_key') || '')
  const [trelloAuthToken, setTrelloAuthToken] = useState(store.getItem('trello_auth_token') || '')

  const searchParams = new URLSearchParams(window.location.search)
  let targets = ''
  if (defaultTargets) {
    targets = defaultTargets
  }
  const urlData = {
    targets: searchParams.getAll('targets').join(',') || targets,
    withClosed: searchParams.get('withClosed') === 'true',
    withoutIsolated: searchParams.get('withoutIsolated') === 'false',
    withoutPrs: searchParams.get('withoutPrs') === 'false',
    withoutExternalDeps: searchParams.get('withoutExternalDeps') === 'false',
    layout: searchParams.get('layout') || '',
  }

  const handleChangeGit = (e) => {
    e.preventDefault()
    const token = event.target.value || ''
    store.setItem('auth_token', token)
    setAuthToken(token)
    // setShowAuthModal(!token)
  }

  const handleChangeTrelloApi = (e) => {
    e.preventDefault()
    const trelloApiToken = event.target.value || ''
    store.setItem('trello_api_key', trelloApiToken)
    setTrelloApiKey(trelloApiToken)
    // setShowAuthModal(!token)
  }

  const handleChangeTrelloAuth = (e) => {
    e.preventDefault()
    const trelloToken = event.target.value || ''
    store.setItem('trello_auth_token', trelloToken)
    setTrelloAuthToken(trelloToken)
    // setShowAuthModal(!token)
  }

  const handleClose = (e) => {
    e.preventDefault()
    setShowAuthModal(false)
  }


  return (
    <StoreProvider context={{ layout: computeLayoutConfig(urlData.layout), urlData }}>
      <div className="page">
        <div className="flex-fill">
          <Menu authToken={authToken} trelloApiKey={trelloApiKey} trelloAuthToken={trelloAuthToken}  handleShowToken={() => setShowAuthModal(true)} urlParams={urlData} />
          
          <Router>
            <Switch>
              <Route exact path="/" component={HomePage} />
              <Route exact path="/githubOAuth" component={GitHubOAuthPage} />
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
        <Modal
          showModal={showAuthModal}
          id="auth-modal"
          size="lg"
        >
          <div className="modal-header">
            <h5 className="modal-title">Enter auth token</h5>
            <button type="button" className="close" data-dismiss="modal" aria-label="Close" onClick={handleClose}>
              <XCircle />
            </button>
          </div>
          <div className="modal-body">
            <p>Enter your auth tokens below.</p>
            <form onSubmit={handleClose}>
              <input type="text" name="authToken" id="authToken" placeholder="Github Auth token" className="form-control" value={authToken} onChange={handleChangeGit} />
              <br />
              <input type="text" name="TrelloApiKey" id="TrelloApiKey" placeholder="Trello Api Key" className="form-control" value={trelloApiKey} onChange={handleChangeTrelloApi} />
              <br />
              <input type="text" name="TrelloAuthToken" id="TrelloAuthToken" placeholder="Trello Auth token" className="form-control" value={trelloAuthToken} onChange={handleChangeTrelloAuth} />
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
