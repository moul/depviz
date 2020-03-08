import React, { useEffect } from 'react'
import { useStore } from '../../hooks/useStore'
import CytoscapeRenderer from './renderers/Cytoscape'
import MermaidRenderer from './renderers/Mermaid'

const VisualizerWrapper = () => {
  let layoutConfig

  const { data, layout } = useStore()
  const { tasks } = data || {}

  const computeLayoutConfig = (layout) => {
    let layoutConfig

    switch (layout) {
      case 'circle':
        layoutConfig = {
          name: 'circle',
          avoidOverlap: true,
        }
        break
      case 'cose':
        layoutConfig = {
          name: 'cose',
          animate: false,
          componentSpacing: 0.5,
          nodeOverlap: 2,
          nodeRepulsion: 0.5,
          nestingFactor: 19,
          gravity: 200,
          numIter: 2000,
          coolingFactor: 0.2,
        }
        break
      case 'breadthfirst':
        layoutConfig = {
          name: 'breadthfirst',
        }
        break
      case 'concentric':
        layoutConfig = {
          name: 'concentric',
        }
        break
      case 'grid':
        layoutConfig = {
          name: 'grid',
          condense: true,
        }
        break
      case 'random':
        layoutConfig = {
          name: 'random',
        }
        break
      case 'cola':
        layoutConfig = {
          name: 'cola',
          animate: false,
          refresh: 1,
          padding: 30,
          maxSimulationTime: 100,
        }
        break
      case 'elk':
        layoutConfig = {
          name: 'elk',
          elk: {
            zoomToFit: true,
            algorithm: 'mrtree',
            separateConnectedComponents: false,
          },
        }
        break
      case 'gantt':
        layoutConfig = {
          name: 'gantt',
        }
        break
      case 'flow':
        layoutConfig = {
          name: 'flow',
        }
        break
      default:
        break
    }

    return layoutConfig
  }

  useEffect(() => {
    layoutConfig = computeLayoutConfig(layout)
  }, [layout])
  console.log('tasks: ', tasks)

  const nodes = []
  const edges = []

  if (tasks) {
    tasks.forEach((task) => {
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
    if (layoutConfig) {
      if (layoutConfig.name === 'gantt' || layoutConfig.name === 'flow') {
        return <MermaidRenderer nodes={nodes} edges={edges} layoutConfig={layoutConfig} />
      }
    }
    return <CytoscapeRenderer nodes={nodes} edges={edges} layoutConfig={layoutConfig} />
  }
  return (
    <div>
      Tasks not found or Repository url is empty
    </div>
  )
}

export default VisualizerWrapper
