import React, { useEffect } from 'react'
import { useForm } from 'react-hook-form'
// import { forEachObjIndexed } from 'ramda'
import { useStore } from '../../../hooks/useStore'
import { generateUrl, updateBrowserHistory } from './utils'
import { fetchDepviz } from '../../../api/depviz'

import './styles.scss'

const Menu = ({
  authToken, showAuth = false, handleShowToken, urlParams = {},
}) => {
  const {
    updateApiData, updateLayout, layout,
  } = useStore()
  const {
    register, getValues, setValue, handleSubmit,
  } = useForm()

  let urlData = urlParams

  useEffect(() => {
    Object.keys(urlData).map((key, index) => {
      if (urlData[key]) {
        setValue(key, urlData[key])
      }
    })
    // forEachObjIndexed(setFormValue, urlData)
    updateLayout(urlData.layout)
    if (urlData.targets) {
      makeAPICall(urlData)
    }
  }, [])


  const makeAPICall = async (data) => {
    const response = await fetchDepviz(`/graph${generateUrl(data)}`)
    updateApiData(response.data, data.layout, data.targets)
    // updateBrowserHistory(url)
  }

  const onSubmit = () => {
    const data = getValues()
    makeAPICall(data)
  }

  const handleLayoutChange = () => {
    const data = getValues()
    updateLayout(data.layout)
    updateBrowserHistory(generateUrl(data))
  }

  const handleCheckboxChange = (e) => {
    const data = getValues()
    // makeAPICall(data)
    urlData = {
      ...urlData,
      ...data,
    }
    urlData.withoutIsolated = !urlData.withoutIsolated
    urlData.withoutPrs = !urlData.withoutPrs
    urlData.withoutExternalDeps = !urlData.withoutExternalDeps

    makeAPICall(urlData)
    updateBrowserHistory(generateUrl(urlData))
  }

  const handleRedraw = () => {
    const cyLayout = window.cy.layout(layout)
    cyLayout.run()
  }

  return (
    <div className="header d-lg-flex p-3">
      <div className="container">
        <form onSubmit={handleSubmit(onSubmit)} className="row align-items-center">
          <div className="col-lg-6 order-lg-first">
            <div className="form-group repo-and-token">
              <label htmlFor="targets" className="form-label">
                <div className="input-group">
                  <input ref={register} type="text" name="targets" id="targets" placeholder="Repository" className="form-control" />
                  <div className="input-group-append">
                    <button type="submit" className="btn btn-primary ml-auto">Generate</button>
                    <button type="button" onClick={handleRedraw} className="btn btn-secondary ml-auto">Redraw</button>
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
                <input ref={register} type="checkbox" name="withClosed" id="withClosed" onChange={handleCheckboxChange} className="custom-control-input" />
                <span className="custom-control-label">Closed</span>
              </label>


              <label htmlFor="withoutIsolated" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutIsolated" id="withoutIsolated" onChange={handleCheckboxChange} className="custom-control-input" />
                <span className="custom-control-label">Isolated</span>
              </label>


              <label htmlFor="withoutPrs" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutPrs" id="withoutPrs" onChange={handleCheckboxChange} className="custom-control-input" />
                <span className="custom-control-label">PRs</span>
              </label>


              <label htmlFor="withoutExternalDeps" className="custom-control custom-checkbox custom-control-inline">
                <input ref={register} type="checkbox" name="withoutExternalDeps" id="withoutExternalDeps" onChange={handleCheckboxChange} className="custom-control-input" />
                <span className="custom-control-label">Ext. Deps</span>
              </label>
            </div>

            <div className="form-group layout-select">
              <label htmlFor="layout">
                <span className="custom-control">Layout:</span>
                <select ref={register} name="layout" id="layout" onChange={handleLayoutChange} className="form-control custom-select selectized">
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
