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
       Issue 1   :done, issue_1, 2019-08-01, 2019-08-02
       Issue 2   :done, issue_2, 2019-08-01, 2019-08-02
       Issue 3   :done, issue_3, 2019-08-01, 2019-08-02
       Issue 4   :done, issue_4, 2019-08-01, 2019-08-02
    `
    const ganttTasks = []
    nodes.forEach((node) => {
      const item = node.data
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
    ganttTemplate += `\t${ganttTasks.join('\n\t\t')}`


    /* const ganttTemplate = `gantt
    dateFormat  YYYY-MM-DD
    title Adding GANTT diagram functionality to mermaid

    section Github Issues
    Issue 1   :done, issue_#1, 2019-07-01, 2019-07-02
    Issue 4   :done, issue_#4, 2019-08-01, 2019-08-02
    Issue 5   :active, issue_#5, after issue_#4 , 2019-08-08, 3d
    Issue 7   :active, issue_#7, after issue_#4 issue_#1 , 2019-08-08, 3d

    Completed task            :done,    des1, 2019-01-06,2019-01-08
    Active task               :active,  des2, 2019-01-09, 3d
    Future task               :         des3, after des2, 5d
    Future task2              :         des4, after des3, 5d
    ` */

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
    let ganttTemplate = `graph ${orientation}
      issue_1(Issue 1)
      issue_2(Issue 2)
      issue_3(Issue 3)
      issue_4(Issue 4)
    `
    const ganttTasks = []
    nodes.forEach((node) => {
      const item = node.data
      let ganttStr = `issue_${item.local_id.replace(repName, '').replace('/', '_')}("${item.description}")`
      if (item.is_depending_on) {
        ganttStr += ' --> '
        for (let i = 0; i < item.is_depending_on.length - 1; i++) {
          const urlArr = item.is_depending_on[i].split('/')
          const issId = urlArr[urlArr.length - 1]
          ganttStr += `issue_${issId.replace('/', '_')}&`
        }
        const urlArr = item.is_depending_on[item.is_depending_on.length - 1].split('/')
        const issId = urlArr[urlArr.length - 1]
        ganttStr += `issue_${issId.replace('/', '_')}`
      }
      ganttTasks.push(ganttStr)
    })
    ganttTemplate += `\t${ganttTasks.join('\n\t\t')}`


    /* const ganttTemplate = `graph TD
    issue_1(Issue 1)
    issue_2(Issue 2)
    issue_3(Issue 3)
    issue_4(Issue 4)
    issue_5(Depends on #4) --> issue_1
    ` */

    const ganttStr = ganttTemplate.toString()
    return ganttStr
  }

  return <div className="mermaid-wrapper" dangerouslySetInnerHTML={{ __html: mermaidGraph }} />
}

export default MermaidRenderer
