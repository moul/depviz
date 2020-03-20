import React from 'react'
import './styles.scss'

class ErrorBoundary extends React.Component {
  constructor(params) {
    super(params)
    this.state = {
      showError: false,
      errMessage: '',
      errStack: '',
    }
  }

  componentDidCatch(error, info) {
    if (error) {
      this.setState({
        showError: true,
        errMessage: error.toString(),
        errStack: info.componentStack.split('\n').map((i) => <p>{i}</p>),
      })
    }
    console.log('error: ', error)
    console.log('info: ', info)
  }

  render() {
    const { showError, errMessage, errStack } = this.state
    if (showError) {
      return (
        <div className="error-stack">
          <div className="error">{errMessage}</div>
          <div className="stack">
            Error stack:
            <br />
            {errStack}
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

export default ErrorBoundary
