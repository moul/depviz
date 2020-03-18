import React, { useEffect, useState } from 'react'
import cytoscape from 'cytoscape'
import cola from 'cytoscape-cola'
import elk from 'cytoscape-elk/src'
import nodeHtmlLabel from 'cytoscape-node-html-label'
import Card from './cardTemplate'
import './card.scss'

import './styles.scss'

const CytoscapeRenderer = ({ nodes, edges, layout }) => {
  const [cyMounted, setCyMount] = useState(false)

  useEffect(() => {
    if (!cyMounted) {
      nodeHtmlLabel(cytoscape)
      cytoscape.use(cola)
      cytoscape.use(elk)

      setCyMount(true)
    }
  }, [cyMounted])

  useEffect(() => {
    const config = {
      container: document.getElementById('cy'),
      elements: [],
      style: [{
        selector: 'node.Issue, node.MergeRequest',
        style: {
          'overlay-padding': '5px',
          'overlay-opacity': 0,
          width: '510px',
          height: '260px',
          shape: 'rectangle',
          'background-color': 'white',
        },
      },
      {
        selector: 'node:parent',
        style: {
          'background-color': 'lightblue',
          opacity: 0.5,
          label: 'data(local_id)',
          padding: 50,
        },
      },
      {
        selector: 'edge',
        style: {
          width: 3,
          'curve-style': 'straight',
        },
      },
      {
        selector: 'edge[arrow]',
        style: {
          'target-arrow-shape': 'data(arrow)',
          'arrow-scale': 5,
        },
      },
      {
        selector: 'edge.hollow',
        style: {
          'target-arrow-fill': 'hollow',
        },
      },
      ],
      layout,
    }

    nodes.forEach((node) => {
      node.group = 'nodes'
      config.elements.push(node)
    })

    const cy = cytoscape(config)
    window.cy = cy

    cy.on('tap', 'node', (evt) => {
      const node = evt.target
      try { // your browser may block popups
        window.open(node.id())
      } catch (e) { // fall back on url change
        window.location.href = node.id()
      }
    })

    cy.nodeHtmlLabel(
      [
        {
          query: 'node.Issue, node.MergeRequest',
          halign: 'center',
          valign: 'center',
          halignBox: 'center',
          valignBox: 'center',
          cssClass: '',
          tpl(data) {
            return Card(data)
          },
        },
      ],
    )

    const edgeMap = {}
    cy.batch(() => {
      edges.forEach((edge) => {
        let isOk = true
        if (cy.getElementById(edge.data.source).empty()) {
          console.warn('missing node', edge.data.source)
          isOk = false
        }
        if (cy.getElementById(edge.data.target).empty()) {
          console.warn('missing node', edge.data.target)
          isOk = false
        }
        if (!isOk) {
          return
        }
        edge.group = 'edges'
        edge.data.id = edge.data.relation + edge.data.source + edge.data.target
        edge.data.arrow = 'triangle'
        if (edge.data.id in edgeMap) {
          console.warn('duplicate edge', edge)
        } else {
          edgeMap[edge.data.id] = edge
          cy.add(edge)
        }
      })
    })
    const cyLayout = cy.layout(layout)
    cyLayout.run()
  }, [layout.name])

  return (
    <div id="cy" />
  )
}

export default CytoscapeRenderer
