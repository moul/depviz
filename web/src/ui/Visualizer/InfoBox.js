import React from 'react'
import { User } from 'react-feather'
import Issue from '../components/icons/Issue'
import Pr from '../components/icons/Pr'
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
  let kindClassIcon = <Issue />
  switch (data.kind) {
    case 'Milestone':
      kindClassIcon = <Pr />
      break
    case 'MergeRequest':
      kindClassIcon = <Pr />
      break
    default:
      break
  }
  const auhorLink = data.has_author
  return (
    <div className="info-box">
      <div className="info-box-wrapper">
        <div className={`info-box-status ${data.state}`} />
        <div className="info-box-title">
          {data.local_id}
          {' '}
          (
          {data.driver}
          )
          <div className="info-box-kind-icon">
            {kindClassIcon}
          </div>
        </div>
        <div className="info-box-body">
          {data.title ? data.title.replace(/"/gi, '\'') : 'No title'}
        </div>
        {auhorLink && (
        <div className="info-box-author-link">
          <User size={16} />
          <a href={`${auhorLink}`} target="_blank" rel="noopener noreferrer">{auhorLink.replace('https://github.com/', '')}</a>
        </div>
        )}
        <div className="info-box-actions">
          <button onClick={openWebLink} className="btn btn-primary ml-auto">View on GitHub</button>
        </div>
      </div>
    </div>
  )
}

export default InfoBox
