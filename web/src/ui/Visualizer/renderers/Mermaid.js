/* eslint-disable react/prop-types */

import React, { useState, useEffect } from 'react'
import { mermaidAPI } from 'mermaid'
import Card from './cardTemplate'
import './card.scss'

const mTemplate = `
graph TD;
  A-->B;
  A-->C;
  B-->D;
  C-->D;
`

const MermaidRenderer = ({ nodes, edges, layoutConfig }) => {
  const [mermaidGraph, setMermaidGraph] = useState('Loading diagram...')

  useEffect(() => {
    /* mermaid.initialize({
      securityLevel: 'loose',
       startOnLoad: true,
       flowchart: {
          useMaxWidth: false,
          htmlLabels: true
      }
    }) */
    mermaidAPI.render('diagram', mTemplate.toString(), (html) => setMermaidGraph(html))
  })

  const renderGanttDiagram = () => {
    setMermaidGraph(mTemplate)
  }

  if (layoutConfig.name === 'gantt') {
    renderGanttDiagram()
  }

  return <div className="mermaid" dangerouslySetInnerHTML={{ __html: mermaidGraph }} />
}

export default MermaidRenderer
