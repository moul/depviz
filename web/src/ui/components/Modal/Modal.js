import React from 'react'

import './styles.scss'

const Modal = ({
  id, size, title = 'Modal title', children,
  handleClose,
  showModal,
}) => (
  <div className={`modal modal-blur fade ${showModal ? 'show' : ''}`} id={`modal-${id}`} tabIndex="-1" role="dialog" aria-hidden="true" style={{ display: showModal ? 'block' : 'none' }}>
    <div className={`modal-dialog ${size ? `modal-${size}` : ''} modal-dialog-centered modal-dialog-scrollable`} role="document">
      <div className="modal-content">
        {children}
      </div>
    </div>
  </div>
)

export default Modal
