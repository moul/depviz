import React, { useEffect } from 'react'
import Stats from 'stats.js'
import { useStore } from '../../hooks/useStore'
import CytoscapeRenderer from './renderers/Cytoscape'
import MermaidRenderer from './renderers/Mermaid'

import ErrorBoundary from '../components/ErrorBoundary/ErrorBoundary'

const showDebug = true // process.env.NODE_ENV === 'development'

const VisualizerWrapper = () => {
  const {
    apiData, layout, repName, isLoadingGraph,
  } = useStore()
  const { tasks } = apiData || {}

  console.log('tasks: ', tasks)

  const nodes = []
  const edges = []

  useEffect(() => {
    if (showDebug) {
      const stats = new Stats()
      stats.showPanel(0) // 0: fps, 1: ms, 2: mb, 3+: custom
      stats.dom.id = 'debug-info'
      // Remove old stats
      const debugInfoBlock = document.getElementById('debug-info')
      if (debugInfoBlock) {
        document.body.removeChild(debugInfoBlock)
      }
      // Add new one
      document.body.appendChild(stats.dom)

      const animate = () => {
        stats.begin()
        // monitored code goes here
        stats.end()
        requestAnimationFrame(animate)
      }
      requestAnimationFrame(animate)
    }
  })

  if (tasks) {
    tasks.forEach((task) => {
      if (!task.kind && !task.state) {
        return
      }
      const node = {
        data: task,
        classes: task.kind,
      }
      switch (task.state) {
        case 'Open':
          switch (task.kind) {
            case 'MergeRequest':
              node.data.card_classes = 'in-progress'
              break
            default:
              node.data.card_classes = 'open'
              break
          }
          break
        case 'Closed':
        case 'Merged':
          node.data.card_classes = 'closed'
          break
        default:
          console.warn('unsupported task.state', task)
          node.data.card_classes = 'error'
      }

      switch (task.kind) {
        case 'Issue':
          node.data.bgcolor = 'lightblue'
          node.data.is_issue = true
          node.data.progress = 0.5
          node.data.card_classes += ' issue'
          break
        case 'Milestone':
          node.data.bgcolor = 'lightgreen'
          node.data.is_milestone = true
          node.data.card_classes += ' milestone'
          break
        case 'MergeRequest':
          node.data.bgcolor = 'purple'
          node.data.is_mergerequest = true
          node.data.card_classes += ' pr'
          break
        default:
          console.warn('unsupported task.kind', task)
          node.data.bgcolor = 'grey'
          node.data.is_issue = true
          node.data.progress = 0
          node.data.card_classes += ' ghost'
      }
      // common
      node.data.nb_parents = 0
      node.data.nb_children = 0
      node.data.nb_related = 0
      node.data.nb_parents += (task.is_blocking !== undefined ? task.is_blocking.length : 0)
      node.data.nb_parents += (task.is_part_of !== undefined ? task.is_part_of.length : 0)
      node.data.nb_children += (task.is_depending_on !== undefined ? task.is_depending_on.length : 0)
      node.data.nb_children += (task.has_part !== undefined ? task.has_part.length : 0)
      node.data.nb_related += (task.is_related_with !== undefined ? task.is_related_with.length : 0)

      // relationships
      if (task.is_depending_on !== undefined) {
        task.is_depending_on.forEach((other) => {
          edges.push({
            data: {
              source: task.id,
              target: other,
              relation: 'is_depending_on',
            },
          })
        })
      }
      if (task.is_blocking !== undefined) {
        task.is_blocking.forEach((other) => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: 'is_depending_on',
            },
          })
        })
      }
      if (task.is_related_with !== undefined) {
        task.is_related_with.forEach((other) => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: 'related_with',
            },
          })
        })
      }
      if (task.is_part_of !== undefined) {
        task.is_part_of.forEach((other) => {
          edges.push({
            data: {
              source: task.id,
              target: other,
              relation: 'part_of',
            },
          })
        })
      }
      if (task.has_part !== undefined) {
        task.has_part.forEach((other) => {
          edges.push({
            data: {
              source: other,
              target: task.id,
              relation: 'part_of',
            },
          })
        })
      }
      if (task.has_owner !== undefined) {
        node.data.parent = task.has_owner
      }
      if (task.has_milestone !== undefined) {
        node.data.parent = task.has_milestone
      }
      // if (task.has_author !== undefined) { edges.push({from: task.has_author, to: task.id}) }
      // if (task.has_assignee !== undefined) { task.has_assignee.forEach(other => edges.push({from: other, to: task.id})) }
      // if (task.has_reviewer !== undefined) { task.has_reviewer.forEach(other => edges.push({from: other, to: task.id})) }
      // if (task.has_label !== undefined) { task.has_label.forEach(other => edges.push({from: other, to: task.id})) }

      nodes.push(node)
    })
  }

  let rendererBlock = null

  const debugInfo = { }
  if (tasks && layout) {
    if (layout.name === 'gantt' || layout.name === 'flow') {
      debugInfo.nodes = nodes.length
      if (layout.name === 'flow') {
        debugInfo.edges = edges.length
      }
      rendererBlock = <MermaidRenderer nodes={nodes} edges={edges} layout={layout} />
    } else {
      debugInfo.nodes = nodes.length
      debugInfo.edges = edges.length
      console.log('layout: ', layout)
      rendererBlock = <CytoscapeRenderer nodes={nodes} edges={edges} layout={layout} />
    }
  } else {
    rendererBlock = (
      <div>
        Tasks not found or Repository url is empty
      </div>
    )
  }

  console.log('is update layouts')
  if (isLoadingGraph) {
    rendererBlock = (
      <div className="error empty">
        Wait a moment. Loading a new graph...
      </div>
    )
  } else if (debugInfo && debugInfo.nodes < 1) {
    rendererBlock = (
      <div className="error empty">
        Rendering issue for link
        {' '}
        <b>{repName}</b>
        Nodes =
        {' '}
        {debugInfo.nodes || 0}
      </div>
    )
  }

  return (
    <div>
      <div className="viz-wrapper card">
        <ErrorBoundary>
          {rendererBlock}
        </ErrorBoundary>
      </div>
      {showDebug && (
      <div className="debug-info">
        <div>
          nodes:
          {' '}
          {debugInfo.nodes || 0}
        </div>
        <div>
          edges:
          {' '}
          {debugInfo.edges || 0}
        </div>
      </div>
      )}
    </div>
  )
}

export default VisualizerWrapper
