import React from "react";
import { useForm } from "react-hook-form";
import "./menu.scss"

const Menu = () => {
  const { register, handleSubmit } = useForm();

  const onSubmit = data => console.log(data)

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <div className="form-group">
        <label htmlFor="targets">Repository:</label>
        <input ref={register} type="text" name="targets" id="targets" />
      </div>

      <div className="form-group">
        <input ref={register} type="checkbox" name="with-closed" id="with-closed" />
        <label htmlFor="with-closed">Closed</label>

        <input ref={register} type="checkbox" name="with-isolated" id="with-isolated" />
        <label htmlFor="with-isolated">Isolated</label>

        <input ref={register} type="checkbox" name="with-prs" id="with-prs" />
        <label htmlFor="with-prs">PRs</label>

        <input ref={register} type="checkbox" name="with-external-deps" id="with-external-deps" />
        <label htmlFor="with-external-deps">Ext. Deps</label>
      </div>

      <div className="form-group">
        <label htmlFor="layout">Layout:</label>
        <select ref={register} name="layout" id="layout">
          <option value="circle">circle</option>
          <option value="cose">cose</option>
          <option value="breadthfirst">breadthfirst</option>
          <option value="concetric">concetric</option>
          <option value="grid">grid</option>
          <option value="random">random</option>
          <option value="cola">cola</option>
          <option value="elk">elk</option>

        </select>
      </div>

      <div className="button-group">
        <button type="button">Redraw</button>
        <button type="submit">Generate</button>
      </div>
    </form>
  )
}

export default Menu;
