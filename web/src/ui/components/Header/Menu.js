import React from 'react'
import { useForm } from 'react-hook-form'
import { fetchDepviz } from '../../../api/depviz'
import { useStore } from '../../../hooks/useStore'

import './styles.scss'

const allowedLayouts = [
  'circle', 'cose', 'breadthfirst', 'concentric', 'grid', 'random', 'cola', 'elk',
  'gantt', 'flow',
]

const Menu = () => {
  const { register, handleSubmit } = useForm()
  const { updateApiData, updateLayout } = useStore()

  const onSubmit = async (data) => {
    const {
      targets,
      withClosed,
      withIsolated,
      withPrs,
      withExternalDeps,
      layout,
    } = data

    // construct url
    const url = `/graph?targets=${targets || 'moul-bot/depviz-test'}&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}&layout=${layout}`

    const response = await fetchDepviz(url)
    updateApiData(response.data, layout, targets)
  }

  return (
    <form onSubmit={handleSubmit((data) => onSubmit(data))}>
      <div className="form-group">
        <label htmlFor="targets">
          Repository:
          <input ref={register} type="text" name="targets" id="targets" defaultValue="" />
        </label>
      </div>

      <div className="form-group">
        <label htmlFor="withClosed">
          <input ref={register} type="checkbox" name="withClosed" id="withClosed" />
          Closed
        </label>

        <label htmlFor="withIsolated">
          <input ref={register} defaultChecked type="checkbox" name="withIsolated" id="withIsolated" />
          Isolated
        </label>

        <label htmlFor="withPrs">
          <input ref={register} defaultChecked type="checkbox" name="withPrs" id="withPrs" />
          PRs
        </label>

        <label htmlFor="withExternalDeps">
          <input ref={register} defaultChecked type="checkbox" name="withExternalDeps" id="withExternalDeps" />
          Ext. Deps
        </label>
      </div>

      <div className="form-group">
        <label htmlFor="layout">
          Layout:
          <select ref={register} name="layout" id="layout" onChange={(e) => updateLayout(e.target.value)}>
            <option value="circle">circle</option>
            <option value="cose">cose</option>
            <option value="breadthfirst">breadthfirst</option>
            <option value="concentric">concentric</option>
            <option value="grid">grid</option>
            <option value="random">random</option>
            <option value="cola">cola</option>
            <option value="elk">elk</option>
            <option value="gantt">gantt</option>
            <option value="flow">flow</option>
          </select>
        </label>
      </div>

      <div className="button-group">
        <button type="submit">Generate</button>
      </div>
    </form>
  )
}

export default Menu
