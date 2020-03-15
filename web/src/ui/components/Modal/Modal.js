import React from 'react'

import './styles.scss'

const Modal = ({
  id, size, title = 'Modal title', children,
  handleClose,
  showModal,
}) => (
  <div className="modal modal-blur fade" id={`modal-${id}`} tabIndex="-1" role="dialog" aria-hidden="true">
    <div className={`modal-dialog ${size ? `modal-${size}` : ''} modal-dialog-centered modal-dialog-scrollable`} role="document">
      <div className="modal-header">
        <h5 className="modal-title">{title}</h5>
        <button type="button" className="close" data-dismiss="modal" aria-label="Close" onClick={handleClose}>
          <i className="fe fe-x" />
        </button>
      </div>

      <div className="modal-content">
        {children}
      </div>

      <div className="modal-footer">
        <button type="button" className="btn btn-secondary mr-auto" data-dismiss="modal">Close</button>
        <button type="button" className="btn btn-primary" data-dismiss="modal">Save changes</button>
      </div>
    </div>
  </div>
)

export default Modal
