import React, { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { forEachObjIndexed } from 'ramda'
import { useStore } from '../../../hooks/useStore'
import { generateUrl, updateBrowserHistory } from './utils'
import { fetchDepviz } from '../../../api/depviz'

import './styles.scss'

const Menu = () => {
  const {
    updateApiData, updateLayout, layout,
  } = useStore()
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
  }, [])


  const makeAPICall = async (data) => {
    const {
      layout,
      targets,
    } = data

    // construct url
    const url = generateUrl(data)

    const response = await fetchDepviz(`/graph${url}`)
    updateApiData(response.data, layout, targets)
    updateBrowserHistory(url)
  }

  const onSubmit = (data) => {
    makeAPICall(data)
  }

  const onRedraw = () => {
    const cyLayout = window.cy.layout(layout)
    cyLayout.run()
  }

  const onLayoutChange = (data) => {
    updateLayout(data.layout)
    updateBrowserHistory(generateUrl(data))
  }

  return (
    <div className="header collapse d-lg-flex p-0">
      <div className="container">
        <div className="row align-items-center">
          <form onSubmit={handleSubmit(onSubmit)}>
            <div className="form-group">
              <label htmlFor="targets" className="form-label">
                Repository:
                <input ref={register} type="text" name="targets" id="targets" className="form-control" />
              </label>
            </div>

            <div className="form-group">

              <label htmlFor="withClosed" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withClosed" id="withClosed" onChange={() => onSubmit(getValues())} className="custom-control-input" />
                <span className="custom-control-label">Closed</span>
              </label>


              <label htmlFor="withIsolated" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} defaultChecked type="checkbox" name="withIsolated" id="withIsolated" onChange={() => onSubmit(getValues())} className="custom-control-input" />
                <span className="custom-control-label">Isolated</span>
              </label>


              <label htmlFor="withPrs" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} defaultChecked type="checkbox" name="withPrs" id="withPrs" onChange={() => onSubmit(getValues())} className="custom-control-input" />
                <span className="custom-control-label">PRs</span>
              </label>


              <label htmlFor="withExternalDeps" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} defaultChecked type="checkbox" name="withExternalDeps" id="withExternalDeps" onChange={() => onSubmit(getValues())} className="custom-control-input" />
                <span className="custom-control-label">Ext. Deps</span>
              </label>
            </div>

            <div className="form-group">
              <label htmlFor="layout">
                Layout:
                <select ref={register} name="layout" id="layout" onChange={() => onLayoutChange(getValues())} className="form-control custom-select selectized">
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
              <button type="submit" className="btn btn-primary ml-auto">Generate</button>
              {/* <button type="button" onClick={onRedraw} className="btn btn-primary ml-auto">Redraw</button> */}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default Menu
