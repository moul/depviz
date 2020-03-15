import React, { useState } from 'react'

import Visualizer from '../../Visualizer'
import Modal from '../../components/Modal/Modal'
import store from '../../../utils/store'

import './styles.scss'

const HomePage = () => {
  const [showAuthModal, setShowAuthModal] = useState(store.getItem('authToken'))
  return (
    <div className="container">
      <div className="row">
        <div className="col-12">
          <Visualizer />
          <Modal
            showModal={showAuthModal}
            handleClose={setShowAuthModal}
            id="auth-modal"
            size="lg"
            title="Enter auth token"
          >
            <div>If you proceed, you will lose all your personal data.</div>
          </Modal>
        </div>
      </div>
    </div>
  )
}

export default HomePage
