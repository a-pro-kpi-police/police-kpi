import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator, kpiSearchText } from '../kpiLocale.js'

export default function KpiCodeCombobox({
  items,
  value,
  onChange,
  includeAll = false,
  allLabel,
  placeholder,
  ariaLabel,
  required = false,
}) {
  const { t, lang } = useI18n()
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const [showAll, setShowAll] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const listboxId = useId()
  const optionRefs = useRef([])

  const selectedItem = useMemo(
    () => items.find((item) => String(item.id) === String(value)),
    [items, value]
  )
  const resolvedAllLabel = allLabel || t('allKpis')
  const resolvedPlaceholder = placeholder || t('searchKpiCodePlaceholder')
  const resolvedAriaLabel = ariaLabel || t('searchKpiCode')
  const filteredItems = useMemo(() => {
    if (showAll || !query.trim()) return items
    const normalizedQuery = query.trim().toLocaleLowerCase()
    return items.filter((item) => kpiSearchText(item).includes(normalizedQuery))
  }, [items, query, showAll])
  const menuItems = useMemo(() => (
    includeAll && (showAll || !query.trim())
      ? [{ id: '', code: resolvedAllLabel, isAll: true }, ...filteredItems]
      : filteredItems
  ), [filteredItems, includeAll, query, resolvedAllLabel, showAll])

  useEffect(() => {
    if (!isOpen) setQuery(selectedItem?.code || '')
  }, [isOpen, selectedItem?.code])

  useEffect(() => {
    if (activeIndex >= menuItems.length) {
      setActiveIndex(Math.max(menuItems.length - 1, 0))
    }
  }, [activeIndex, menuItems.length])

  useEffect(() => {
    if (isOpen) {
      optionRefs.current[activeIndex]?.scrollIntoView({ block: 'nearest' })
    }
  }, [activeIndex, isOpen])

  const chooseItem = (item) => {
    onChange(item.isAll ? '' : String(item.id))
    setQuery(item.isAll ? '' : item.code)
    setIsOpen(false)
    setShowAll(false)
    setActiveIndex(0)
  }

  const openAll = () => {
    setShowAll(true)
    setIsOpen(true)
    const selectedIndex = items.findIndex((item) => String(item.id) === String(value))
    setActiveIndex(selectedIndex >= 0 ? selectedIndex + (includeAll ? 1 : 0) : 0)
  }

  const handleKeyDown = (event) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (!isOpen) {
        openAll()
        return
      }
      setActiveIndex((index) => Math.min(index + 1, Math.max(menuItems.length - 1, 0)))
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (!isOpen) {
        openAll()
        return
      }
      setActiveIndex((index) => Math.max(index - 1, 0))
    } else if (event.key === 'Enter' && isOpen) {
      event.preventDefault()
      if (menuItems[activeIndex]) chooseItem(menuItems[activeIndex])
    } else if (event.key === 'Escape') {
      setIsOpen(false)
      setShowAll(false)
      setQuery(selectedItem?.code || '')
    }
  }

  return (
    <div
      className="kpi-combobox"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          setIsOpen(false)
          setShowAll(false)
          setQuery(selectedItem?.code || '')
        }
      }}
    >
      <input
        className="kpi-combobox-input"
        type="text"
        role="combobox"
        value={query}
        onChange={(event) => {
          setQuery(event.target.value)
          onChange('')
          setShowAll(false)
          setIsOpen(true)
          setActiveIndex(0)
        }}
        onFocus={(event) => event.target.select()}
        onClick={openAll}
        onKeyDown={handleKeyDown}
        placeholder={includeAll ? resolvedAllLabel : resolvedPlaceholder}
        aria-label={resolvedAriaLabel}
        aria-autocomplete="list"
        aria-controls={listboxId}
        aria-expanded={isOpen}
        aria-activedescendant={
          isOpen && menuItems[activeIndex]
            ? `${listboxId}-${menuItems[activeIndex].id || 'all'}`
            : undefined
        }
        autoComplete="off"
        spellCheck={false}
        required={required}
      />
      <button
        type="button"
        className="kpi-combobox-toggle"
        aria-label={resolvedAriaLabel}
        aria-expanded={isOpen}
        onClick={() => {
          if (isOpen) {
            setIsOpen(false)
            setShowAll(false)
          } else {
            openAll()
          }
        }}
      >
        ▾
      </button>
      {isOpen && (
        <div
          id={listboxId}
          className="kpi-combobox-list"
          role="listbox"
          aria-label={resolvedAriaLabel}
        >
          {!menuItems.length ? (
            <div className="kpi-combobox-empty" role="status">
              {t('noKpiMatches')}
            </div>
          ) : menuItems.map((item, index) => (
            <div
              id={`${listboxId}-${item.id || 'all'}`}
              key={item.id || 'all'}
              ref={(node) => { optionRefs.current[index] = node }}
              className={`kpi-combobox-option${index === activeIndex ? ' active' : ''}${item.isAll ? ' is-all' : ''}`}
              role="option"
              aria-selected={item.isAll ? !value : String(item.id) === String(value)}
              onMouseEnter={() => setActiveIndex(index)}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => chooseItem(item)}
            >
              <strong>{item.code}</strong>
              {!item.isAll && (
                <>
                  <span>{kpiIndicator(item, lang)}</span>
                  <small>{item.points_per_unit} {t('points')}/{item.unit}</small>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
