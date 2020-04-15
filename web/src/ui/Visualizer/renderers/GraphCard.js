import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import Issue from '../../components/icons/Issue'
import Pr from '../../components/icons/Pr'
import Comments from '../../components/icons/Comments'
import Github from '../../components/icons/Github'

import './card.scss'

const GraphCard = (data, type) => {
  // console.log('data: ', data)
  let cardClasses = data.card_classes
  if (type === 'mermaid') {
    cardClasses = data.state ? data.state.toLowerCase() : ''
  }
  let kindClassIcon = <div className="cy-icon icon-issue"><Issue /></div>
  switch (data.kind) {
    case 'Milestone':
      kindClassIcon = <div className="cy-icon icon-pr"><Pr /></div>
      break
    case 'MergeRequest':
      kindClassIcon = <div className="cy-icon icon-pr"><Pr /></div>
      break
    default:
      break
  }

  const cardTemplate = (
    <div className={`cy-card issue ${cardClasses}`}>
      <div className="b-left">
        {kindClassIcon}
      </div>
      <div className="b-body">
        <div className="b-body-top">
          <div className="id">{data.local_id}</div>
          <div className="icons">
            <div className="cy-icon icon-comments">
              <Comments />
            </div>
            <div className="cy-icon icon-github">
              <Github />
            </div>
            <div className="cy-icon avatar" />
          </div>
        </div>
        <div className="b-body-middle">
          <div className="title">
            {data.title.replace(/"/gi, '\'')}
          </div>
        </div>
        {/*
        <div class='b-body-bottom'>
          <span class='tag red'>high priority</span>
          <span class='tag yellow'>warning</span>
          <span class='tag blue'>cool</span>
        </div>
      */}
      </div>
      <div className="b-right">
        <span className={`circle ${data.nb_parents > 0 ? 'red' : 'green'}`}>{data.nb_parents}</span>
        <span className={`circle ${data.nb_children > 0 ? 'red' : 'green'}`}>{data.nb_children}</span>
        <span className="circle grey">
          {data.nb_related}
        </span>
      </div>
      {/*
      <div class='b-progress'>
        <div class='b-progress-left'>
          70%
        </div>
        <div class='b-progress-right'>
          <div class='bar bar-bg'>
            <div class='bar bar-progress' style='width:70%;'></div>
          </div>
        </div>
      </div>
      */}
    </div>
  )

  let cardTemplateString = renderToStaticMarkup(cardTemplate)
  if (type === 'mermaid') {
    cardTemplateString = cardTemplateString.replace(/(\r\n|\n|\r)/gm, '').replace(/> *</g, '><').replace(/"/gi, '\'')
    // .replace(/"/gi, '\"').replace(/'/gi, '\'')
    // cardTemplate = cardTemplate.replace(/'/gm, '\\'')
  }
  return cardTemplateString
}

export default GraphCard
