import { useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/internationalisation',
  component: Internationalisation,
})

interface Locale {
  tag: string
  label: string
  dir: 'ltr' | 'rtl'
  greeting: string
  // Expansion is the translation of the same sentence, and the point is that
  // it is not the same length.
  sentence: string
}

const LOCALES: Locale[] = [
  { tag: 'en-GB', label: 'English', dir: 'ltr', greeting: 'Good morning', sentence: 'Delete this item' },
  { tag: 'de-DE', label: 'Deutsch', dir: 'ltr', greeting: 'Guten Morgen', sentence: 'Dieses Element löschen' },
  { tag: 'ar-EG', label: 'العربية', dir: 'rtl', greeting: 'صباح الخير', sentence: 'احذف هذا العنصر' },
  { tag: 'ja-JP', label: '日本語', dir: 'ltr', greeting: 'おはようございます', sentence: 'この項目を削除' },
  { tag: 'hi-IN', label: 'हिन्दी', dir: 'ltr', greeting: 'सुप्रभात', sentence: 'इस आइटम को हटाएँ' },
]

// The same instant and the same amount everywhere, so only the formatting
// differs and the differences are the whole subject.
const INSTANT = new Date(Date.UTC(2026, 2, 4, 15, 30, 0))
const AMOUNT = 1234567.891

const CURRENCY: Record<string, string> = {
  'en-GB': 'GBP', 'de-DE': 'EUR', 'ar-EG': 'EGP', 'ja-JP': 'JPY', 'hi-IN': 'INR',
}

// Identical to the eye, different byte for byte. Written as escapes rather
// than as literal text so the difference is visible in the source, and typed
// as string because TypeScript otherwise narrows them to distinct literal
// types and refuses the very comparison this page exists to demonstrate.
const NFC: string = 'caf\u00e9' // e-acute as one code point
const NFD: string = 'cafe\u0301' // plain e, then a combining acute accent

// One grapheme built from several code points joined by zero-width joiners.
const FAMILY = '👩‍👩‍👧‍👦'

function Internationalisation() {
  const [locale, setLocale] = useState(LOCALES[0]!)
  const [typed, setTyped] = useState('')

  return (
    <ChallengePage id="internationalisation">
      <div className="flex flex-wrap gap-2" data-testid="language-switcher">
        {LOCALES.map((option) => (
          <button
            key={option.tag}
            data-testid={`locale-${option.tag}`}
            className={option.tag === locale.tag ? 'primary' : ''}
            onClick={() => setLocale(option)}
          >
            {option.label}
          </button>
        ))}
      </div>

      <section
        className="mt-6 rounded-lg border border-line p-4"
        lang={locale.tag}
        dir={locale.dir}
        data-testid="locale-panel"
        data-locale={locale.tag}
        data-dir={locale.dir}
      >
        <p className="m-0 text-lg" data-testid="greeting">
          {locale.greeting}
        </p>
        <button className="mt-3" data-testid="translated-action">
          {locale.sentence}
        </button>
        <p className="mt-2 text-sm text-muted">
          The button's label is <b data-testid="label-length">{locale.sentence.length}</b>{' '}
          characters here. In English it is {LOCALES[0]!.sentence.length}.
        </p>
      </section>

      <table className="results mt-6">
        <tbody>
          <tr>
            <th>number</th>
            <td data-testid="format-number">{AMOUNT.toLocaleString(locale.tag)}</td>
          </tr>
          <tr>
            <th>currency</th>
            <td data-testid="format-currency">
              {AMOUNT.toLocaleString(locale.tag, {
                style: 'currency',
                currency: CURRENCY[locale.tag] ?? 'USD',
              })}
            </td>
          </tr>
          <tr>
            <th>date</th>
            <td data-testid="format-date">
              {new Intl.DateTimeFormat(locale.tag, { dateStyle: 'short' }).format(INSTANT)}
            </td>
          </tr>
          <tr>
            <th>one item</th>
            <td data-testid="plural-one">
              {new Intl.PluralRules(locale.tag).select(1)}
            </td>
          </tr>
          <tr>
            <th>two items</th>
            <td data-testid="plural-two">
              {new Intl.PluralRules(locale.tag).select(2)}
            </td>
          </tr>
        </tbody>
      </table>

      <section className="mt-6">
        <h2 className="stage__heading">Text that compares as unequal while looking identical</h2>
        <table className="results">
          <tbody>
            <tr>
              <th>composed</th>
              <td data-testid="nfc">{NFC}</td>
            </tr>
            <tr>
              <th>decomposed</th>
              <td data-testid="nfd">{NFD}</td>
            </tr>
            <tr>
              <th>equal as written</th>
              <td data-testid="naive-equal">{String(NFC === NFD)}</td>
            </tr>
            <tr>
              <th>equal once normalised</th>
              <td data-testid="normalised-equal">{String(NFC.normalize() === NFD.normalize())}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">One emoji, several code points</h2>
        <p className="text-3xl" data-testid="family">
          {FAMILY}
        </p>
        <table className="results">
          <tbody>
            <tr>
              <th>length</th>
              <td data-testid="family-length">{FAMILY.length}</td>
            </tr>
            <tr>
              <th>code points</th>
              <td data-testid="family-codepoints">{[...FAMILY].length}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">Typing in the current script</h2>
        <input
          data-testid="script-input"
          dir={locale.dir}
          lang={locale.tag}
          className="w-80 rounded-md border border-line bg-sunken px-2 py-1"
          placeholder={locale.greeting}
          value={typed}
          onChange={(event) => setTyped(event.target.value)}
        />
        <p className="mt-2 text-sm text-muted">
          Round-tripped: <b data-testid="typed-back">{typed || '(nothing)'}</b>, length{' '}
          <b data-testid="typed-length">{typed.length}</b>
        </p>
      </section>
    </ChallengePage>
  )
}
