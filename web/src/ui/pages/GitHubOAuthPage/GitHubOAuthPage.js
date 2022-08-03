import React from 'react';
import {getToken} from "../../../api/depviz";
import store from "../../../utils/store";
import { useHistory } from "react-router-dom";

const GitHubOAuthPage = () => {
  const history = useHistory();
  // request of GitHub OAuth
  const getTokenOAuth = async (code) => {
    const { data } = await getToken('/token', code)
    store.setItem('auth_token', data.access_token)
  }

  React.useEffect(() => {
    const code = new URLSearchParams(window.location.search).get('code')
    if (code) {
      getTokenOAuth(code).then(() => {
        history.push('/')
      });
    } else {
      history.push('/')
    }
  })

  return (
    <div>GitHubOAuth</div>
  )
}

export default GitHubOAuthPage
