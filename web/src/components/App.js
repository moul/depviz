/* eslint-disable import/no-named-as-default */
import { Route, Switch, BrowserRouter as Router} from "react-router-dom";
import PropTypes from "prop-types";
import React from "react";
import { hot } from "react-hot-loader";
import Provider from "../store/provider";
import HomePage from "./pages/HomePage";
import Menu from "./menu"

class App extends React.Component {
  render() {
    return (
      <Provider>
        <Router>
          <Menu />
          <div>
            <Switch>
              <Route exact path="/" component={HomePage} />
            </Switch>
          </div>
        </Router>
      </Provider>
    );
  }
}

App.propTypes = {
  children: PropTypes.element
};

export default hot(module)(App);
