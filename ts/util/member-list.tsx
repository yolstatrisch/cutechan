/**
 * Member list widget.
 */

import { Component, h } from "preact";
import { _ } from "../lang";

interface MemberListProps {
  myName: string;
  members: string[];
  disabled?: boolean;
  onChange: (members: string[]) => void;
}

class MemberList extends Component<MemberListProps, {}> {
  public shouldComponentUpdate(nextProps: MemberListProps) {
    return this.props.members !== nextProps.members;
  }
  public render({ members, disabled }: MemberListProps) {
    return (
      <ul class="member-list">
        {members.map((name) =>
          <li class="member-list-item" onClick={() => this.handleRemove(name)}>
            {name}
          </li>,
        )}
        <li class="member-list-newitem">
          <input
            class="member-list-name"
            placeholder={_("Add name")}
            title={_("Enter to add")}
            disabled={disabled}
            onKeyDown={this.handleNameKey}
          />
        </li>
      </ul>
    );
  }
  private isValid(name: string): boolean {
    // TODO(Kagami): Validate name chars and highlight invalid inputs?
    return (
      name.length >= 1
      && name.length <= 20
      && name !== this.props.myName
      && !this.props.members.includes(name)
    );
  }
  private handleRemove = (name: string) => {
    if (this.props.disabled) return;
    if (name === this.props.myName) return;
    const members = this.props.members.filter((n) => n !== name);
    this.props.onChange(members);
  }
  private handleNameKey = (e: KeyboardEvent) => {
    if (e.keyCode === 13) {
      const nameEl = e.target as HTMLInputElement;
      const name = nameEl.value;
      if (!this.isValid(name)) return;
      const members = this.props.members.concat(name);
      this.props.onChange(members);
      nameEl.value = "";
    }
  }
}

export default MemberList;
