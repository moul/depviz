import React from 'react';
const Trigger = ({ triggerText, buttonRef, showModal }) => {
  return (
    <button
      className="btn btn-primary ml-auto"
      ref={buttonRef}
      onClick={showModal}
    >
      {triggerText}
    </button>
  );
};
export default Trigger;
