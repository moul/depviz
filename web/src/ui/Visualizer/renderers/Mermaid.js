import React, { useState, useEffect } from 'react'
import { mermaidAPI } from 'mermaid'
import { useStore } from '../../../hooks/useStore'
import GraphCard from './GraphCard'

import './styles.scss'

const isDev = process.env.NODE_ENV === 'development'

const MermaidRenderer = ({ nodes, layout, handleInfoBox }) => {
  const { repName } = useStore()
  const [mermaidGraph, setMermaidGraph] = useState('Loading diagram...')
  const [mermaidOrientation, setMermaidOrientation] = useState('TB')
  const [graphInfo, setGraphInfo] = useState('')

  const clickCardAway = (e) => {
    console.log('tap on background')
    const node = e.target
    if (node.classList.contains('active')) {
      // inside click
      return
    }
    // Check clicks on infobox
    if (typeof node.className === 'string') {
      if (node.className.includes('info-box')) {
        // inside click
        return
      }
    }

    // hack to avoid click on mermaid cards
    if (node.classList.contains('b-body')
      || node.classList.contains('b-right')
      || node.classList.contains('b-left')
      || node.classList.contains('title')
      || node.classList.contains('cy-card')
      || node.classList.contains('circle')
    ) {
      // inside click
      return
    }
    const parentMermaidDOM = document.getElementById('mermaid-graph-id')
    // Get all active elements
    const activeNodes = parentMermaidDOM.getElementsByClassName('active')
    for (let i = 0; i < activeNodes.length; i += 1) {
      if (activeNodes[i].classList.contains('active')) {
        activeNodes[i].classList.remove('active')
      }
    }
    // outside click
    handleInfoBox(null, false)
  }

  useEffect(() => {
    window.clickOnCardEvent = (params) => {
      // Flow graph send issue Id as a param only
      let nodeData = ''

      // Remove all active nodes first
      const parentMermaidDOM = document.getElementById('mermaid-graph-id')
      // Get all active elements
      const activeNodes = parentMermaidDOM.getElementsByClassName('active')
      for (let i = 0; i < activeNodes.length; i += 1) {
        if (activeNodes[i].classList.contains('active')) {
          activeNodes[i].classList.remove('active')
        }
      }

      try {
        nodeData = JSON.parse(params).data

        // Find node and set active class
        const nodeElem = document.querySelector(`[id="${nodeData.issueId}"]`)
        if (!nodeElem.classList.contains('active')) {
          nodeElem.classList.add('active')
        }
        // Find node text and set active class (for Gantt and Timeline graphs)
        const nodeTextElem = document.querySelector(`[id="${nodeData.issueId}-text"]`)
        if (nodeTextElem) {
          if (!nodeTextElem.classList.contains('active')) {
            nodeTextElem.classList.add('active')
          }
        }
      } catch (err) {
        console.log(err)
        // Flow graph processing
        for (let i = 0; i < nodes.length; i += 1) {
          if (nodes[i].data.local_id) {
            const issueId = `issue${nodes[i].data.local_id.replace(`${repName}#`, '').replace(/\//gi, '_').replace(/#/gi, '_')}`
            if (issueId === params) {
              const item = nodes[i].data
              nodeData = {
                ...item,
                issueId,
              }
            }
          }
        }
        // Find node and set active class
        const parentElem = document.getElementById(`${nodeData.issueId}`)
        const cardElem = parentElem.getElementsByClassName('cy-card')
        if (cardElem[0]) {
          if (!cardElem[0].classList.contains('active')) {
            cardElem[0].classList.add('active')
          }
        }
      }

      handleInfoBox(nodeData)
    }

    mermaidAPI.initialize({
      securityLevel: 'loose',
      maxTextSize: 1000000, // TODO: optimize node label rendering
      mermaid: {
        // startOnLoad: false,
      },
      flowchart: {
        useMaxWidth: true,
        curve: 'cardinal',
      },
    })

    // add when mounted
    document.addEventListener('click', clickCardAway)
    // return function to be called when unmounted
    return () => {
      document.removeEventListener('click', clickCardAway)
    }
  })

  useEffect(() => {
    if (layout.name === 'gantt') {
      mermaidAPI.render(
        'gantt',
        renderGanttTemplate(),
        (svgHtml, bindFunctions) => {
          // console.log('svgHtml: ', svgHtml)
          setMermaidGraph(svgHtml)
          // Hack to bind events to rendered graph
          setTimeout(() => {
            const elem = document.querySelector('[id="mermaid-graph-id"]')
            bindFunctions(elem)
          }, 1000)
        },
      )
    } else if (layout.name === 'flow') {
      mermaidAPI.render(
        'diagram',
        renderFlowTemplate(),
        (svgHtml, bindFunctions) => {
          setMermaidGraph(svgHtml)
          // Hack to bind events to rendered graph
          setTimeout(() => {
            const elem = document.querySelector('[id="mermaid-graph-id"]')
            bindFunctions(elem)
          }, 1000)
        },
      )
    } else if (layout.name === 'timeline') {
      mermaidAPI.render(
        'gantt',
        renderTimelineTemplate(),
        (svgHtml, bindFunctions) => {
          setMermaidGraph(svgHtml)
          // Hack to bind events to rendered graph
          setTimeout(() => {
            const elem = document.querySelector('[id="mermaid-graph-id"]')
            bindFunctions(elem)
          }, 1000)
        },
      )
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
      // const cardTpl = GraphCard(item)
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
            const createdDateStr = item.created_at.split('T')[0]
            const completedDateStr = item.completed_at ? item.completed_at.split('T')[0] : '7d'
            ganttTasks.push(`Missing node issue${issId.replace(/\//gi, '_')}   :done, issue${issId.replace('/', '_')},${createdDateStr}, ${completedDateStr}`)
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
      const issData = {
        ...item,
        issueId,
      }
      ganttClickTasks.push(`\n\r\tclick ${issueId} call clickOnCardEvent("{"data": ${JSON.stringify(issData)}}")`)
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
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }
      const issueId = `issue${item.local_id.replace(`${repName}#`, '').replace(/\//gi, '_').replace(/#/gi, '_')}`
      const cardTpl = GraphCard(item, 'mermaid')
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
      flowClickEvents.push(`click ${issueId.replace(/\//gi, '_')} clickOnCardEvent`)
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

  const renderTimelineTemplate = () => {
    let timelineTemplate = `gantt
    dateFormat  YYYY-MM-DD
    title ${repName}

    section Github Issues
    `
    const timelineTasks = []
    const timelineClickTasks = []
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }

      if (item.state !== 'Closed' && item.state !== 'Merged') {
        return
      }

      // item state
      let status = ''
      switch (item.state) {
        case 'Closed':
        case 'Merged':
          status = 'done'
          break
        default:
          break
      }

      const issueId = `issue${item.local_id.replace(`${repName}#`, '').replace(/\//gi, '_').replace(/#/gi, '_')}`
      // const cardTpl = GraphCard(item)
      let timelineStr = `${issueId} [${item.title}]   `
      if (!item.is_depending_on) {
        timelineStr += `:${status}, ${issueId}`
      } else {
        timelineStr += `:${issueId}`
      }

      if (item.is_depending_on) {
        timelineStr += ', after'
        for (let i = 0; i < item.is_depending_on.length; i += 1) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          const issIdStr = `issue${issId.replace(/\//gi, '_')}`
          // Check missing nodes
          let nodeInStack = false
          for (let j = 0; j < timelineTasks.length; j += 1) {
            const ganttItem = timelineTasks[j]
            if (ganttItem.includes(issIdStr)) {
              nodeInStack = true
              break
            }
          }

          if (!nodeInStack || timelineTasks.length === 0) {
            // Add missing node first
            const createdDateStr = item.created_at.split('T')[0]
            const completedDateStr = item.completed_at ? item.completed_at.split('T')[0] : '7d'
            timelineTasks.push(`Missing node issue${issId.replace(/\//gi, '_')}   :done, issue${issId.replace('/', '_')}, ${createdDateStr}, ${completedDateStr}`)
            timelineStr += ` issue${issId.replace(/\//gi, '_')}`
          } else {
            timelineStr += ` issue${issId.replace(/\//gi, '_')}`
          }
        }
      }

      if (!item.is_depending_on) {
        const createdDateStr = item.created_at.split('T')[0]
        const completedDateStr = item.completed_at.split('T')[0]
        timelineStr += `, ${createdDateStr}, ${completedDateStr}`
      } else {
        const completedDateStr = item.completed_at.split('T')[0]
        timelineStr += `, ${completedDateStr}`
      }
      timelineTasks.push(timelineStr)
      const issData = {
        ...item,
        issueId,
      }
      timelineClickTasks.push(`\n\r\tclick ${issueId} call clickOnCardEvent("{"data": ${JSON.stringify(issData)}}")`)
    })

    // Remove uplicates
    const noDupsGanttTasks = [...new Set(timelineTasks)]

    timelineTemplate += `${noDupsGanttTasks.join('\n\t')}`
    // Add click links
    timelineTemplate += `\n\r\t%% Click events${timelineClickTasks.join('\t')}`

    const timelineStr = timelineTemplate.toString()
    setGraphInfo(timelineStr)
    return timelineStr
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
        <div id="mermaid-graph-id" className="mermaid-graph" dangerouslySetInnerHTML={{ __html: mermaidGraph }} />
      </div>
      {isDev && (
      <div className="mermaid-graph-info">
        <h3>Graph layout (for debug)</h3>
        {graphInfo.split('\n').map((node, index) => <p key={index} dangerouslySetInnerHTML={{ __html: node.replace(/\\t/gi, '&nbsp;').replace(/\s/gi, '&nbsp;') }} />)}
      </div>
      )}
    </div>
  )
}

export default MermaidRenderer
