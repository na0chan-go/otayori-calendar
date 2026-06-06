import { inject, provide, type InjectionKey } from 'vue'
import type { useOtayoriCalendar } from './useOtayoriCalendar'

export type OtayoriCalendarContext = ReturnType<typeof useOtayoriCalendar>

const otayoriCalendarKey: InjectionKey<OtayoriCalendarContext> = Symbol('otayori-calendar')

export function provideOtayoriCalendar(context: OtayoriCalendarContext) {
  provide(otayoriCalendarKey, context)
}

export function useOtayoriCalendarContext() {
  const context = inject(otayoriCalendarKey)
  if (!context) throw new Error('Otayori Calendar context is not provided')
  return context
}
