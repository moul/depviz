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
      ganttStr += `:${status}, issue_${item.local_id.replace(repName, '').replace('/', '_')}`
      if (item.is_depending_on) {
        ganttStr += ', after '
        for (let i = 0; i < item.is_depending_on.length; i++) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          ganttStr += `issue_${issId.replace('/', '_')} `
        }
      }
      const dateStr = item.created_at.split('T')[0]
      ganttStr += `, ${dateStr}, 7d`
      ganttTasks.push(ganttStr)
    })
    ganttTemplate += `${ganttTasks.join('\n\t')}`

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
    let flowTemplate = `graph ${orientation}`

    const flowTasks = []
    nodes.forEach((node) => {
      const item = node.data
      if (!item.local_id) {
        return
      }
      const issId = `issue_${item.local_id.replace(repName, '').replace('/', '_')}`
      let flowStr = `${issId}("${issId}")`
      if (item.is_depending_on) {
        flowStr += ' --> '
        for (let i = 0; i < item.is_depending_on.length - 1; i++) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          flowStr += `issue_${issId.replace('/', '_')}&`
        }
        const urlArr = item.is_depending_on[item.is_depending_on.length - 1].split('/')
        const issId = urlArr[urlArr.length - 1]
        flowStr += `issue_${issId.replace('/', '_')}(issue_#${issId})`
      }
      flowTasks.push(flowStr)
    })
    flowTemplate += `\t${flowTasks.join('\n\t\t')}`


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
