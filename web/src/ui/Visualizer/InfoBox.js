import React from 'react'
import './infoBox.scss'

const InfoBox = ({ data }) => {
  console.log('data: ', data)
  const openWebLink = () => {
    try { // your browser may block popups
      window.open(data.id)
    } catch (e) { // fall back on url change
      window.location.href = data.id
    }
  }
  return (
    <div className="info-box">
      <div className="info-box-wrapper">
        <div className="info-box-title">
          {data.id}
        </div>
        <div className="info-box-body">
          {data.title.replace(/"/gi, '\'')}
        </div>
        <div className="info-box-actions">
          <button onClick={openWebLink} className="btn btn-primary ml-auto">View on GitHub</button>
        </div>
      </div>
    </div>
  )
}

export default InfoBox
