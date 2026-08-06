import { LitElement, css, html, type TemplateResult } from 'lit'

/**
 * Three open shadow roots, one inside the next. Lit attaches an open root by
 * default, so the tree is reachable — the point is that reaching it takes a
 * deliberate traversal, not a document-wide query.
 */

const frame = css`
  :host {
    display: block;
    border: 1px dashed var(--border, #d8d5ce);
    border-radius: 10px;
    padding: 1rem 1.1rem;
    font-family: var(--font, system-ui, sans-serif);
    color: var(--text, #1c1b19);
  }

  .depth {
    margin: 0 0 0.85rem;
    font-family: var(--mono, monospace);
    font-size: 0.6875rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--muted, #6f6b63);
  }
`

class ShadowOuter extends LitElement {
  static override styles = frame

  override render(): TemplateResult {
    return html`
      <p class="depth">Shadow root 1 of 3</p>
      <slot name="label"></slot>
      <pg-shadow-middle></pg-shadow-middle>
    `
  }
}

class ShadowMiddle extends LitElement {
  static override styles = frame

  override render(): TemplateResult {
    return html`
      <p class="depth">Shadow root 2 of 3</p>
      <pg-shadow-inner></pg-shadow-inner>
    `
  }
}

class ShadowInner extends LitElement {
  static override styles = [
    frame,
    css`
      label {
        display: block;
        font-size: 0.875rem;
        margin-bottom: 0.7rem;
      }

      input {
        display: block;
        margin-top: 0.3rem;
        width: min(22rem, 100%);
        font: inherit;
        padding: 0.4rem 0.6rem;
        border: 1px solid var(--border, #d8d5ce);
        border-radius: 7px;
        background: var(--surface, #fff);
        color: inherit;
      }

      button {
        font: inherit;
        font-size: 0.9375rem;
        padding: 0.4rem 0.9rem;
        border-radius: 7px;
        border: 1px solid var(--accent, #b4541e);
        background: var(--accent, #b4541e);
        color: #fff;
        cursor: pointer;
      }

      .echo {
        margin: 0.8rem 0 0;
        font-size: 0.875rem;
        color: var(--muted, #6f6b63);
      }
    `,
  ]

  static override properties = { typed: { state: true } }
  declare typed: string

  constructor() {
    super()
    this.typed = ''
  }

  override render(): TemplateResult {
    return html`
      <p class="depth">Shadow root 3 of 3</p>
      <label>
        Message
        <input
          data-testid="inner-input"
          .value=${this.typed}
          @input=${(event: Event) => {
            this.typed = (event.target as HTMLInputElement).value
          }}
        />
      </label>
      <button data-testid="inner-submit" @click=${this.send}>Send it outwards</button>
      <p class="echo" data-testid="inner-echo">
        ${this.typed === '' ? 'nothing typed yet' : this.typed}
      </p>
    `
  }

  // composed: true is what lets the event escape all three roots. Without it
  // the listener on the document never fires, which is its own lesson.
  private send(): void {
    this.dispatchEvent(
      new CustomEvent('pg-shadow-submit', {
        detail: { value: this.typed },
        bubbles: true,
        composed: true,
      }),
    )
  }
}

customElements.define('pg-shadow-outer', ShadowOuter)
customElements.define('pg-shadow-middle', ShadowMiddle)
customElements.define('pg-shadow-inner', ShadowInner)

// The light DOM listens for the event that crossed the boundary, giving the
// page a target that needs no shadow traversal at all.
document.addEventListener('pg-shadow-submit', (event) => {
  const { value } = (event as CustomEvent<{ value: string }>).detail
  const echo = document.querySelector('[data-testid="shadow-echo"]')
  const count = document.querySelector('[data-testid="shadow-submit-count"]')

  if (echo) echo.textContent = value === '' ? '(empty)' : value
  if (count) count.textContent = String(Number(count.textContent ?? '0') + 1)
})
