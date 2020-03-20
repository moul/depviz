import React, { useState, useEffect } from 'react'
import { mermaidAPI } from 'mermaid'
import { useStore } from '../../../hooks/useStore'

const MermaidRenderer = ({ nodes, layout }) => {
  const { repName } = useStore()
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
    if (layout.name === 'gantt') {
      mermaidAPI.render('gantt', renderGanttTemplate(), (html) => setMermaidGraph(html))
    } else if (layout.name === 'flow') {
      mermaidAPI.render('diagram', renderFlowTemplate(), (html) => setMermaidGraph(html))
    }
  }, [layout.name])

  const renderGanttTemplate = () => {
    let ganttTemplate = `gantt
    dateFormat  YYYY-MM-DD
    title ${repName}

    section Github Issues
    `
    const ganttTasks = []
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }
      let ganttStr = `${item.title}   `
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

      if (!item.is_depending_on) {
        ganttStr += `:${status}, issue${item.local_id.replace(`${repName}#`, '').replace('/', '_')}`
      } else {
        ganttStr += `:issue${item.local_id.replace(`${repName}#`, '').replace('/', '_')}`
      }

      if (item.is_depending_on) {
        ganttStr += ', after'
        for (let i = 0; i < item.is_depending_on.length; i++) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          const issIdStr = `issue${issId.replace('/', '_')}`
          // Check missing nodes
          let nodeInStack = false
          for (let j = 0; j < ganttTasks.length; j++) {
            const ganttItem = ganttTasks[j]
            if (ganttItem.includes(issIdStr)) {
              nodeInStack = true
              break
            }
          }

          if (!nodeInStack || ganttTasks.length === 0) {
            // Add missing node first
            ganttTasks.push(`Missing node issue${issId.replace('/', '_')}   :done, issue${issId.replace('/', '_')}, 2019-08-06, 7d`)
            ganttStr += ` issue${issId.replace('/', '_')}`
          } else {
            ganttStr += ` issue${issId.replace('/', '_')}`
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
    })

    // Remove uplicates
    const noDupsGanttTasks = [...new Set(ganttTasks)]

    ganttTemplate += `${noDupsGanttTasks.join('\n\t')}`

    const ganttStr = ganttTemplate.toString()
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
  const renderFlowTemplate = (orientation = 'TB') => {
    let flowTemplate = `graph ${orientation}\n\r`

    const flowTasks = []
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }
      const issId = `issue${item.local_id.replace(`${repName}#`, '').replace('/', '_')}`
      let flowStr = `${issId}("${issId}")`
      if (item.is_depending_on) {
        flowStr += ' --> '
        for (let i = 0; i < item.is_depending_on.length - 1; i++) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          const issIdStr = `issue${issId.replace('/', '_')}&`
          // Check missing nodes
          let nodeInStack = false
          for (let j = 0; j < flowTasks.length; j++) {
            const flowItem = flowTasks[j]
            if (flowItem.includes(issIdStr)) {
              nodeInStack = true
              break
            }
          }

          if (!nodeInStack || flowTasks.length === 0) {
            // Add missing node first
            flowTasks.push(`issue${issId.replace('/', '_')}(missing issue${issId})\n\rstyle issue${issId.replace('/', '_')} fill:#ddd`)
            flowStr += `issue${issId.replace('/', '_')}&`
          } else {
            flowStr += `issue${issId.replace('/', '_')}&`
          }
        }
        const urlArr = item.is_depending_on[item.is_depending_on.length - 1].split('/')
        const issId = urlArr[urlArr.length - 1]
        flowStr += `issue${issId.replace('/', '_')}(issue${issId})`
      }
      flowTasks.push(flowStr)
    })
    flowTemplate += `\t${flowTasks.join('\n\t')}`


    /* const ganttTemplate = `graph TD
    issue_1(Issue 1)
    issue_2(Issue 2)
    issue_3(Issue 3)
    issue_4(Issue 4)
    issue_5(Depends on #4) --> issue_1
    ` */

    const flowStr = flowTemplate.toString()
    return flowStr
  }

  return <div className="mermaid-wrapper" dangerouslySetInnerHTML={{ __html: mermaidGraph }} />
}

export default MermaidRenderer
