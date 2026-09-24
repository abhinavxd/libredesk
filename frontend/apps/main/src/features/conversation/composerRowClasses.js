export const ROW_CLASS = 'relative space-y-0 border-b border-border px-3'
export const ROW_LABEL_CLASS = 'w-12 shrink-0 text-sm font-normal text-muted-foreground'
export const ROW_INPUT_CLASS =
  'h-9 flex-1 min-w-0 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0 focus:ring-0 placeholder:text-muted-foreground/60 data-[placeholder]:text-muted-foreground/60'
export const ROW_COMBOBOX_CLASS =
  'h-9 flex-1 min-w-0 border-0 bg-transparent px-0 font-normal shadow-none hover:bg-transparent'
export const ROW_COMBOBOX_EMPTY_CLASS = `${ROW_COMBOBOX_CLASS} text-muted-foreground/60`
export const ROW_MESSAGE_CLASS = 'pb-2 pl-14'

export const isUnset = (value) => !value || value === 'none'
