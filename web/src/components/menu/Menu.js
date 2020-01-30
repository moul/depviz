import React from "react";
import "./menu.scss"

const Menu = () => {
  return (
    <section>
      <div className="form-group">
        <label htmlFor="repositoryInput">Repository:</label>
        <input type="text" name="repository-input" id="repositoryInput" />
      </div>

      <div className="form-group">
        <input type="checkbox" value="closed" name="closed-checkbox" id="closed" />
        <label htmlFor="closed">Closed</label>

        <input type="checkbox" value="isolated" name="isolated-checkbox" id="isolated" />
        <label htmlFor="isolated">Isolated</label>

        <input type="checkbox" value="prs" name="prs-checkbox" id="prs" />
        <label htmlFor="prs">PRs</label>

        <input type="checkbox" value="extDeps" name="extDeps-checkbox" id="extDeps" />
        <label htmlFor="extDeps">Ext. Deps</label>
      </div>

      <div className="form-group">
        <label htmlFor="visulaizationType">Visulatization type:</label>
        <select name="visualization" id="visulaizationType">
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
    </section>
  )
}

export default Menu;
