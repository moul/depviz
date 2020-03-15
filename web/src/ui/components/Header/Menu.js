import React, { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { forEachObjIndexed } from 'ramda'
import { useStore } from '../../../hooks/useStore'
import { generateUrl, updateBrowserHistory } from './utils'
import { fetchDepviz } from '../../../api/depviz'

import './styles.scss'

const Menu = ({ showAuth = false, handleShowToken }) => {
  const {
    updateApiData, updateLayout,
  } = useStore()
  const {
    register, getValues, setValue, handleSubmit,
  } = useForm()
  const searchParams = new URLSearchParams(window.location.search)
  const urlData = {
    targets: searchParams.getAll('targets').join(',') || '',
    withClosed: searchParams.get('withClosed') || '',
    withIsolated: searchParams.get('withIsolated') || '',
    withPrs: searchParams.get('withPrs') || '',
    withExternalDeps: searchParams.get('withoutExternal-deps') || '',
    layout: searchParams.get('layout') || '',
  }

  useEffect(() => {
    const formValues = getValues()
    if (formValues.targets) {
      urlData.targets = formValues.targets.legnth > 1 ? formValues.targets.join(',') : formValues.targets
      setValue('targets', urlData.targets)
    }
    if (formValues.withClosed) {
      urlData.withClosed = formValues.withClosed
    }
    if (formValues.withIsolated) {
      urlData.withIsolated = formValues.withIsolated
    }
    if (formValues.withPrs) {
      urlData.withPrs = formValues.withPrs
    }
    if (formValues.withExternalDeps) {
      urlData.withExternalDeps = formValues.withExternalDeps
    }
    if (formValues.layout) {
      urlData.layout = formValues.layout
    }

    const setFormValue = (value, key) => {
      if (value) {
        setValue(key, value)
      }
    }

    forEachObjIndexed(setFormValue, urlData)

    if (urlData.targets) {
      makeAPICall(urlData)
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

  const onLayoutChange = (data) => {
    updateLayout(data.layout)
    updateBrowserHistory(generateUrl(data))
  }

  return (
    <div className="header d-lg-flex p-3">
      <div className="container">
        <form onSubmit={handleSubmit(onSubmit)} className="row align-items-center">
          <div className="col-lg-5 order-lg-first">
            <div className="form-group repo-and-token">
              <label htmlFor="targets" className="form-label">
                <div className="input-group">
                  <input ref={register} type="text" name="targets" id="targets" placeholder="Repository" className="form-control" />
                  <div className="input-group-append">
                    <button type="submit" className="btn btn-primary ml-auto">Generate</button>
                    {/* <button type="button" onClick={onRedraw} className="btn btn-primary ml-auto">Redraw</button> */}
                  </div>
                </div>
              </label>
              <a onClick={handleShowToken} className="btn">
                {authToken ? 'Change token' : '+ Add token'}
              </a>
            </div>

          </div>
          <div className="col-lg ml-right">
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

            <div className="form-group layout-select">
              <label htmlFor="layout">
                <span className="custom-control">Layout:</span>
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
          </div>
        </form>
      </div>
    </div>
  )
}

export default Menu
