/* eslint-disable import/no-named-as-default */
import { Route, Switch, BrowserRouter as Router } from 'react-router-dom'
import React from 'react'
import { hot } from 'react-hot-loader'
import { StoreProvider } from './hooks/useStore'
import HomePage from './ui/pages/HomePage/HomePage'
import Menu from './ui/components/Header/Menu'

const App = () => (
  <StoreProvider>
    <Router>
      <Menu />
      <div>
        <Switch>
          <Route exact path="/" component={HomePage} />
        </Switch>
      </div>
    </Router>
  </StoreProvider>
)

export default hot(module)(App)
