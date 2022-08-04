import React, { Component } from 'react';
import { Modal } from './Modal';
import TriggerButton from './TriggerButton';
import {fetchDepviz} from "../../../../api/depviz";
import {generateUrl} from "../../Header/utils";
export class Container extends Component {
  state = { isShown: false };

  Metadata = (owner, repo, id, newMetadata) => {
    fetchDepviz(`/github/issue/add/metadata${generateUrl({
      owner: owner,
      repo: repo,
      id: id,
      metadata: newMetadata,
    })}`)
  };

  onSubmit = (event) => {
    event.preventDefault(event);
    const data = this.props.githubURI.split('/');
    if (event.target.time !== undefined) {
      this.Metadata(data[3], data[4], data[6], event.target.time.value);
    }
    if (event.target.depend !== undefined) {
      isNaN(event.target.depend.value) ?
      this.Metadata(data[3], data[4], data[6], "depends on " + event.target.depend.value) :
      this.Metadata(data[3], data[4], data[6], "depends on %23" + event.target.depend.value);
    }
    if (event.target.block !== undefined) {
      isNaN(event.target.block.value) ?
      this.Metadata(data[3], data[4], data[6], "blocks " + event.target.block.value) :
      this.Metadata(data[3], data[4], data[6], "blocks %23" + event.target.block.value);
    }
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
