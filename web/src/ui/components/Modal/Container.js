import React, { Component } from 'react';
import { Modal } from './Assign';
import TriggerButton from './TriggerButton';
import {fetchDepviz} from "../../../api/depviz";
import {generateUrl} from "../Header/utils";
export class Container extends Component {
  state = { isShown: false };

    Assign = (owner, repo, id, assignee) => {
    fetchDepviz(`/github/assign${generateUrl({
      owner: owner,
      repo: repo,
      id: id,
      assignee: assignee,
    })}`)
  };

  onSubmit = (event) => {
    event.preventDefault(event);
    console.log(event.target.name.value);
    const data = this.props.githubURI.split('/');
    this.Assign(data[3], data[4], data[6], event.target.name.value);
    this.closeModal();
  };
  showModal = () => {
    this.setState({ isShown: true }, () => {
      this.closeButton.focus();
    });
    this.toggleScrollLock();
  };
  closeModal = () => {
    this.setState({ isShown: false });
    this.TriggerButton.focus();
    this.toggleScrollLock();
  };
  onKeyDown = (event) => {
    if (event.keyCode === 27) {
      this.closeModal();
    }
  };
  onClickOutside = (event) => {
    if (this.modal && this.modal.contains(event.target)) return;
    this.closeModal();
  };

  toggleScrollLock = () => {
    document.querySelector('html').classList.toggle('scroll-lock');
  };
  render() {
    return (
      <React.Fragment>
        <TriggerButton
          showModal={this.showModal}
          buttonRef={(n) => (this.TriggerButton = n)}
          triggerText={this.props.triggerText}
        />
        {this.state.isShown ? (
          <Modal
            onSubmit={this.onSubmit}
            modalRef={(n) => (this.modal = n)}
            buttonRef={(n) => (this.closeButton = n)}
            closeModal={this.closeModal}
            onKeyDown={this.onKeyDown}
            onClickOutside={this.onClickOutside}
          />
        ) : null}
      </React.Fragment>
    );
  }
}

export default Container;
