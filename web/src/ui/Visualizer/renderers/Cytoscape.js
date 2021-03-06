import React, { useEffect, useState } from 'react'
import cytoscape from 'cytoscape'
import cola from 'cytoscape-cola'
import elk from 'cytoscape-elk/src'
import nodeHtmlLabel from 'cytoscape-node-html-label'
import { useStore } from '../../../hooks/useStore'
import GraphCard from './GraphCard'
import './styles.scss'

const CytoscapeRenderer = ({
  nodes, edges, layout, handleInfoBox,
}) => {
  const { urlData } = useStore()
  const [cyMounted, setCyMount] = useState(false)

  useEffect(() => {
    if (!cyMounted) {
      // Register nodeHtmlLabel extension (if not exists already)
      if (window.cy) {
        if (!window.cy.nodeHtmlLabel) {
          console.log('register nodeHtmlLabel')
          nodeHtmlLabel(cytoscape)
        }
      }
      // Register Cola and Elk extensions
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
          width: '494px',
          height: '224px',
          shape: 'rectangle',
          padding: 0,
          'background-color': 'white',
        },
      },
      {
        selector: 'node:parent',
        style: {
          'background-color': 'lightblue',
          opacity: 0.5,
          label: 'data(local_id)',
          padding: 20,
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
          'arrow-scale': 3,
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
      const newNode = node
      newNode.group = 'nodes'
      config.elements.push(newNode)
    })

    const cy = cytoscape(config)
    window.cy = cy

    cy.on('tap', (event) => {
      // target holds a reference to the originator
      // of the event (core or element)
      const evtTarget = event.target
      if (evtTarget === cy) {
        console.log('tap on background')
        const nodes = cy.nodes()
        nodes.removeClass('active')
        nodes.style({
          'border-width': '1px',
          'border-color': '#ffffff',
        })
        handleInfoBox(null, false)
      } else {
        console.log('tap on some element')
        const nodeData = evtTarget.data()
        if (!nodeData.card_classes.includes('active') && !evtTarget.hasClass('active')) {
          // evtTarget.data('card_classes', `${nodeData.card_classes} active`)
          evtTarget.style({
            'border-width': '3px',
            'border-radius': '8px',
            'border-color': '#0043ff',
          })
          evtTarget.addClass('active')
        }
        // Double check highlighted card
        /* const nodeElem = document.querySelector(`[id="${issueId}"]`)
        if (nodeElem) {
          if (!nodeElem.classList.contains('active')) {
            nodeElem.classList.add('active')
          }
        } */
        handleInfoBox(nodeData)
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
            return GraphCard(data)
          },
        },
      ],
    )

    cy.batch(() => {
      const edgeMap = {}
      edges.forEach((edge) => {
        const newEdge = edge
        // let isOk = true
        if (cy.getElementById(newEdge.data.source).empty()) {
          console.warn('missing node', newEdge.data.source)
          // isOk = false
          const newNode = {}
          newNode.group = 'nodes'
          newNode.classes = 'Issue'
          newNode.data = {
            id: newEdge.data.source,
            created_at: new Date(),
            updated_at: new Date(),
            local_id: newEdge.data.source.replace('https://github.com/', '').replace('/issues/', '#'),
            html_id: newEdge.data.source.replace('https://github.com/', '').replace('/issues/', '').replace(/\//gi, '_').replace(/#/gi, '_'),
            kind: 'Issue',
            title: 'Ghost issue',
            driver: 'GitHub',
            state: 'Missing',
            card_classes: 'ghost issue',
            bgcolor: 'grey',
            is_issue: true,
            progress: 0.5,
            nb_parents: 0,
            nb_children: 0,
            nb_related: 0,
            parent: undefined,
          }
          // config.elements.push(newNode)
          cy.add(newNode)
        }
        if (cy.getElementById(newEdge.data.target).empty()) {
          console.warn('missing node', newEdge.data.target)
          // isOk = false
          const newNode = {}
          newNode.group = 'nodes'
          newNode.classes = 'Issue'
          newNode.data = {
            id: newEdge.data.target,
            created_at: new Date(),
            updated_at: new Date(),
            local_id: newEdge.data.target.replace('https://github.com/', '').replace('/issues/', '#'),
            html_id: newEdge.data.target.replace('https://github.com/', '').replace('/issues/', '').replace(/\//gi, '_').replace(/#/gi, '_'),
            kind: 'Issue',
            title: 'Ghost issue',
            driver: 'GitHub',
            state: 'Missing',
            card_classes: 'ghost issue',
            bgcolor: 'grey',
            is_issue: true,
            progress: 0.5,
            nb_parents: 0,
            nb_children: 0,
            nb_related: 0,
            parent: undefined,
          }
          cy.add(newNode)
          // config.elements.push(newNode)
        }
        // if (!isOk) {
        //   return
        // }
        newEdge.group = 'edges'
        newEdge.data.id = `edge_${newEdge.data.relation}_${newEdge.data.source}_${newEdge.data.target}`
        newEdge.data.arrow = 'triangle'
        if (newEdge.data.id in edgeMap) {
          console.warn('duplicate edge', newEdge)
        } else {
          edgeMap[newEdge.data.id] = newEdge
          cy.add(newEdge)
        }
      })
    })

    cy.layout(layout).run()
  }, [
    edges.length,
    nodes.length,
    layout.name,
    !!urlData.withClosed,
    !!urlData.withoutIsolated,
    !!urlData.withoutPrs,
    !!urlData.withoutExternalDeps,
  ])

  console.log('Cytoscape rendering')
  return (
    <div id="cy" />
  )
}

export default CytoscapeRenderer
