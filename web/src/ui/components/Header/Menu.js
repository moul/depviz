import React, { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { forEachObjIndexed } from 'ramda'
import { useStore } from '../../../hooks/useStore'
import { generateUrl, updateBrowserHistory } from './utils'
import { fetchDepviz } from '../../../api/depviz'

import './styles.scss'

const Menu = () => {
  const { updateApiData, updateLayout, layout } = useStore()
  const {
    register, getValues, setValue, handleSubmit,
  } = useForm()
  const searchParams = new URLSearchParams(window.location.search)

  useEffect(() => {
    const urlData = {
      targets: searchParams.getAll('targets').join(',') || undefined,
      withClosed: searchParams.get('withClosed') || undefined,
      withIsolated: searchParams.get('withIsolated') || undefined,
      withPrs: searchParams.get('withPrs') || undefined,
      withExternalDeps: searchParams.get('withoutExternal-deps') || undefined,
      layout: searchParams.get('layout') || undefined,
    }

    const setFormValue = (value, key) => {
      if (value) {
        setValue(key, value)
      }
    }

    forEachObjIndexed(setFormValue, urlData)

    if (urlData.targets) {
      try {
        makeAPICall(urlData)
      } catch (error) {
        throw error
      }
    }
  })

  const onSubmit = (data) => {
    makeAPICall(data)
  }

  const onRedraw = (layout) => {
    updateLayout(layout)
    // const layoutConfig = computeLayoutConfig(layout)
    const cyLayout = window.cy.layout(layoutConfig)
    cyLayout.run()
  }

  const makeAPICall = async (data) => {
    const {
      layout,
      targets,
    } = data

    // construct url
    const url = generateUrl(data)

    try {
      const response = await fetchDepviz(`/graph${url}`)
      updateApiData(response.data, layout, targets)
      updateBrowserHistory(url)
    } catch (error) {
      throw error
    }
  }

  const onLayoutChange = (data) => {
    updateLayout(data.layout)
    updateBrowserHistory(generateUrl(data))
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <div className="form-group">
        <label htmlFor="targets">
          Repository:
          <input ref={register} type="text" name="targets" id="targets" />
        </label>
      </div>

      <div className="form-group">

        <label htmlFor="withClosed">
          Closed
          <input ref={register} type="checkbox" name="withClosed" id="withClosed" onChange={() => onSubmit(getValues())} />
        </label>


        <label htmlFor="withIsolated">
          Isolated
          <input ref={register} defaultChecked type="checkbox" name="withIsolated" id="withIsolated" onChange={() => onSubmit(getValues())} />
        </label>


        <label htmlFor="withPrs">
          PRs
          <input ref={register} defaultChecked type="checkbox" name="withPrs" id="withPrs" onChange={() => onSubmit(getValues())} />
        </label>


        <label htmlFor="withExternalDeps">
          Ext. Deps
          <input ref={register} defaultChecked type="checkbox" name="withExternalDeps" id="withExternalDeps" onChange={() => onSubmit(getValues())} />
        </label>
      </div>

      <div className="form-group">
        <label htmlFor="layout">
          Layout:
          <select ref={register} name="layout" id="layout" onChange={() => onLayoutChange(getValues())}>
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
        <button type="button" onClick={() => onRedraw(getValues().layout)}>Redraw</button>
      </div>
    </form>
  )
}

export default Menu
