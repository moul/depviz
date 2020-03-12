import React, { useEffect, useState } from 'react'
import cytoscape from 'cytoscape'
import cola from 'cytoscape-cola'
import elk from 'cytoscape-elk/src'
import nodeHtmlLabel from 'cytoscape-node-html-label'
import Card from './cardTemplate'
import './card.scss'

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
      }, {
        selector: 'node:parent',
        style: {
          'background-color': 'lightblue',
          opacity: 0.5,
          label: 'data(local_id)',
          padding: 50,
        },
      }],
      layout,
    }

    nodes.forEach((node) => {
      node.group = 'nodes'
      config.elements.push(node)
    })

    const cy = cytoscape(config)

    cy.on('tap', 'node', function () {
      try { // your browser may block popups
        window.open(this.data('id'))
      } catch (e) { // fall back on url change
        window.location.href = this.data('id')
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
        if (edge.data.id in edgeMap) {
          console.warn('duplicate edge', edge)
        } else {
          edgeMap[edge.data.id] = edge
          cy.add(edge)
        }
      })
    })
  }, [layout.name])

  return (
    <div id="cy" />
  )
}

export default CytoscapeRenderer
