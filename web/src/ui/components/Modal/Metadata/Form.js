import React from 'react';

export const Form = ({ onSubmit }) => {
  return (
    <form onSubmit={onSubmit}>
      <div className="form-group">
        <label htmlFor="name">Time of the task</label>
        <input className="form-control" id="time" />
      </div>
      <div className="form-group">
        <label htmlFor="name">Depends on which id task ?</label>
        <input className="form-control" id="depend" />
      </div>
      <div className="form-group">
        <label htmlFor="name">This task blocks which task ?</label>
        <input className="form-control" id="block" />
      </div>
      <div className="form-group">
        <button className="form-control btn btn-primary" type="submit">
          Submit
        </button>
      </div>
    </form>
  );
};
export default Form;
