import React, { useState, useEffect } from 'react'
import { mermaidAPI } from 'mermaid'
import { useStore } from '../../../hooks/useStore'
import MermaidCard from './MermaidCard'

import './styles.scss'

const MermaidRenderer = ({ nodes, layout }) => {
  const { repName } = useStore()
  const [mermaidGraph, setMermaidGraph] = useState('Loading diagram...')
  const [mermaidOrientation, setMermaidOrientation] = useState('TB')
  const [graphInfo, setGraphInfo] = useState('')

  useEffect(() => {
    mermaidAPI.initialize({
      securityLevel: 'loose',
      maxTextSize: 1000000, // TODO: optimize node label rendering
      flowchart: {
        useMaxWidth: true,
        htmlLabels: true,
        curve: 'cardinal',
      },
    })
  })

  useEffect(() => {
    if (layout.name === 'gantt') {
      mermaidAPI.render('gantt', renderGanttTemplate(), (html) => setMermaidGraph(html))
    } else if (layout.name === 'flow') {
      mermaidAPI.render('diagram', renderFlowTemplate(), (html) => setMermaidGraph(html))
    }
  }, [layout.name, nodes.length, mermaidOrientation])

  const renderGanttTemplate = () => {
    let ganttTemplate = `gantt
    dateFormat  YYYY-MM-DD
    title ${repName}

    section Github Issues
    `
    const ganttTasks = []
    const ganttClickTasks = []
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }

      // item state
      let status = ''
      switch (item.state) {
        case 'Open':
          status = 'active'
          break
        case 'Closed':
        case 'Merged':
          status = 'done'
          break
        default:
          break
      }

      const issueId = `issue${item.local_id.replace(`${repName}#`, '').replace(/\//gi, '_').replace(/#/gi, '_')}`
      // const cardTpl = MermaidCard(item)
      let ganttStr = `${issueId} [${item.title}]   `
      if (!item.is_depending_on) {
        ganttStr += `:${status}, ${issueId}`
      } else {
        ganttStr += `:${issueId}`
      }

      if (item.is_depending_on) {
        ganttStr += ', after'
        for (let i = 0; i < item.is_depending_on.length; i += 1) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          const issIdStr = `issue${issId.replace(/\//gi, '_')}`
          // Check missing nodes
          let nodeInStack = false
          for (let j = 0; j < ganttTasks.length; j += 1) {
            const ganttItem = ganttTasks[j]
            if (ganttItem.includes(issIdStr)) {
              nodeInStack = true
              break
            }
          }

          if (!nodeInStack || ganttTasks.length === 0) {
            // Add missing node first
            ganttTasks.push(`Missing node issue${issId.replace(/\//gi, '_')}   :done, issue${issId.replace('/', '_')}, 2019-08-06, 7d`)
            ganttStr += ` issue${issId.replace(/\//gi, '_')}`
          } else {
            ganttStr += ` issue${issId.replace(/\//gi, '_')}`
          }
        }
      }

      if (!item.is_depending_on) {
        const dateStr = item.created_at.split('T')[0]
        ganttStr += `, ${dateStr}, 7d`
      } else {
        ganttStr += ', 7d'
      }
      ganttTasks.push(ganttStr)
      ganttClickTasks.push(`\n\r\tclick ${issueId} href "${item.id}"`)
    })

    // Remove uplicates
    const noDupsGanttTasks = [...new Set(ganttTasks)]

    ganttTemplate += `${noDupsGanttTasks.join('\n\t')}`
    // Add click links
    ganttTemplate += `\n\r\t%% Click events${ganttClickTasks.join('\t')}`

    const ganttStr = ganttTemplate.toString()
    setGraphInfo(ganttStr)
    return ganttStr
  }

  /*
    params:
      orientation: String
        possible values
        TB - top bottom
        BT - bottom top
        RL - right left
        LR - left right
        TD - same as TB
  */
  const renderFlowTemplate = () => {
    let flowTemplate = `graph ${mermaidOrientation}\n\r`

    const flowTasks = []
    const flowClickEvents = []
    /* const callback = (e) => {
      const node = e.target
      try { // your browser may block popups
        window.open(node.id())
      } catch (e) { // fall back on url change
        window.location.href = node.id()
      }
    } */
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }
      const issueId = `issue${item.local_id.replace(`${repName}#`, '').replace(/\//gi, '_').replace(/#/gi, '_')}`
      const cardTpl = MermaidCard(item)
      let flowStr = `${issueId}("${cardTpl}")`
      if (item.is_depending_on) {
        flowStr += ' --> '
        for (let i = 0; i < item.is_depending_on.length - 1; i += 1) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          const issIdStr = `issue${issId.replace(/\//gi, '_')}&`
          // Check missing nodes
          let nodeInStack = false
          for (let j = 0; j < flowTasks.length; j += 1) {
            const flowItem = flowTasks[j]
            if (flowItem.includes(issIdStr)) {
              nodeInStack = true
              break
            }
          }

          if (!nodeInStack || flowTasks.length === 0) {
            // Add missing node first
            flowTasks.push(`issue${issId.replace(/\//gi, '_')}("${cardTpl}")`)
            flowStr += `issue${issId.replace(/\//gi, '_')}&`
          } else {
            flowStr += `issue${issId.replace(/\//gi, '_')}&`
          }
        }
        const urlArr = item.is_depending_on[item.is_depending_on.length - 1].split('/')
        const issId = urlArr[urlArr.length - 1]
        flowStr += `issue${issId.replace('/', '_')}("${cardTpl}")`
      }
      flowTasks.push(flowStr)
      flowClickEvents.push(`click ${issueId.replace(/\//gi, '_')} "${item.id}" "Open ${issueId.replace(/\//gi, '_')} link"`)
    })
    flowTemplate += `\t${flowTasks.join('\n\t')}`
    // Add click links
    flowTemplate += `\n\r\t%% Click events\n\r\t${flowClickEvents.join('\n\r\t')}`

    const flowStr = flowTemplate.toString()
    setGraphInfo(flowStr)
    return flowStr
  }

  const handleMermaidOrientation = (orientation) => () => {
    setMermaidOrientation(orientation)
  }

  return (
    <div className="mermaid-wrapper">
      {layout.name === 'flow' && (
      <div className="selectgroup mermaid-actions">
        <div>
          Flow direction:
          <div>
            <button onClick={handleMermaidOrientation('TB')} className={mermaidOrientation === 'TB' ? 'btn btn-primary ml-auto' : 'btn btn-secondary ml-auto'}>TB</button>
            <button onClick={handleMermaidOrientation('BT')} className={mermaidOrientation === 'BT' ? 'btn btn-primary ml-auto' : 'btn btn-secondary ml-auto'}>BT</button>
            <button onClick={handleMermaidOrientation('RL')} className={mermaidOrientation === 'RL' ? 'btn btn-primary ml-auto' : 'btn btn-secondary ml-auto'}>RL</button>
            <button onClick={handleMermaidOrientation('LR')} className={mermaidOrientation === 'LR' ? 'btn btn-primary ml-auto' : 'btn btn-secondary ml-auto'}>LR</button>
          </div>
        </div>
      </div>
      )}
      <br />
      <div className="mermaid-graph-wrapper">
        <div className="mermaid-graph" dangerouslySetInnerHTML={{ __html: mermaidGraph }} />
      </div>
      <div className="mermaid-graph-info">
        <h3>Graph layout (for debug)</h3>
        {graphInfo.split('\n').map((node, index) => <p key={index} dangerouslySetInnerHTML={{ __html: node.replace(/\\t/gi, '&nbsp;').replace(/\s/gi, '&nbsp;') }} />)}
      </div>
    </div>
  )
}

export default MermaidRenderer
