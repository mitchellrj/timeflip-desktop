export const defaultSettings = {
  communicationTimeoutSeconds: 10,
  commandTimeoutSeconds: 5,
  initialRetrySeconds: 15,
  mediumRetrySeconds: 60,
  longRetrySeconds: 300,
  offlineAfterSeconds: 120,
  offlineAfterFailures: 3,
  weekStartsOn: 'locale',
};

export const defaultReportPreset = 'today';
export const defaultHistoryPageSize = 20;

export const defaultTapSettings = {
  deviceID: '',
  threshold: 20,
  limit: 10,
  latency: 5,
  window: 30,
  confirmedOnDevice: false,
};

export const defaultLEDSettings = {
  deviceID: '',
  brightnessPercent: 50,
  blinkSeconds: 10,
  confirmedOnDevice: false,
};

export function configToSettings(config = {}) {
  const policy = config.reconnectPolicy || {};
  return {
    communicationTimeoutSeconds: durationToSeconds(config.communicationTimeout, defaultSettings.communicationTimeoutSeconds),
    commandTimeoutSeconds: durationToSeconds(config.commandTimeout, defaultSettings.commandTimeoutSeconds),
    initialRetrySeconds: durationToSeconds(policy.initialRetryInterval, defaultSettings.initialRetrySeconds),
    mediumRetrySeconds: durationToSeconds(policy.mediumRetryInterval, defaultSettings.mediumRetrySeconds),
    longRetrySeconds: durationToSeconds(policy.longRetryInterval, defaultSettings.longRetrySeconds),
    offlineAfterSeconds: durationToSeconds(policy.offlineAfterDuration, defaultSettings.offlineAfterSeconds),
    offlineAfterFailures: Number(policy.offlineAfterFailures || defaultSettings.offlineAfterFailures),
    weekStartsOn: weekStartsOnValue(config.weekStartsOn, defaultSettings.weekStartsOn),
  };
}

export function durationToSeconds(value, fallback) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return fallback;
  }
  return Math.round(numeric / 1_000_000_000);
}

export function secondsToDuration(value) {
  return Math.max(0, Number(value || 0)) * 1_000_000_000;
}

export function reportPeriodForPreset(preset = defaultReportPreset, now = new Date(), options = {}) {
  const current = toDate(now);
  const locale = options.locale || defaultLocale();
  const timeZone = options.timeZone || defaultTimeZone();
  const weekStartsOn = weekStartsOnValue(options.weekStartsOn, 'locale');
  const today = localMidnight(current);
  let from = today;
  let to = addCalendarDays(today, 1);
  if (preset === 'yesterday') {
    from = addCalendarDays(today, -1);
    to = today;
  } else if (preset === 'this-week') {
    const weekStart = weekStartDay(weekStartsOn, locale);
    const diff = (today.getDay() - weekStart + 7) % 7;
    from = addCalendarDays(today, -diff);
    to = addCalendarDays(from, 7);
  } else if (preset === 'this-month') {
    from = new Date(today.getFullYear(), today.getMonth(), 1);
    to = new Date(today.getFullYear(), today.getMonth() + 1, 1);
  } else if (preset !== 'today') {
    preset = defaultReportPreset;
  }
  return { preset, from, to, locale, timeZone, weekStartsOn };
}

export function validateCustomPeriod(fromValue, toValue, options = {}) {
  if (!fromValue || !toValue) {
    return { valid: false, error: 'Start and end are required.' };
  }
  const from = toDate(fromValue);
  const to = toDate(toValue);
  if (!Number.isFinite(from.getTime()) || !Number.isFinite(to.getTime())) {
    return { valid: false, error: 'Start and end must be valid dates.' };
  }
  if (from >= to) {
    return { valid: false, error: 'Start must be before end.' };
  }
  return {
    valid: true,
    period: {
      preset: 'custom',
      from,
      to,
      locale: options.locale || defaultLocale(),
      timeZone: options.timeZone || defaultTimeZone(),
      weekStartsOn: weekStartsOnValue(options.weekStartsOn, 'locale'),
    },
  };
}

export function toControllerPeriodRequest(period, now = new Date()) {
  return {
    from: toDate(period.from),
    to: toDate(period.to),
    now: toDate(now),
  };
}

export function compactDuration(seconds) {
  const safeSeconds = Math.max(0, Math.floor(Number(seconds || 0)));
  if (safeSeconds < 60) {
    return `${safeSeconds}s`;
  }
  const minutes = Math.floor(safeSeconds / 60);
  if (minutes < 60) {
    return `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours < 24) {
    return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
  }
  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
}

export function summaryBarPercent(row, rows = []) {
  const maxSeconds = Math.max(0, ...rows.map((item) => Number(item.activeSeconds || 0)));
  if (!maxSeconds) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round((Number(row?.activeSeconds || 0) / maxSeconds) * 100)));
}

export function historyOverlapLabel(session, from, to) {
  const started = toDate(session?.startedAt);
  const ended = session?.endedAt ? toDate(session.endedAt) : null;
  const rangeFrom = toDate(from);
  const rangeTo = toDate(to);
  const startsBefore = Number.isFinite(started.getTime()) && started < rangeFrom;
  const endsAfter = ended ? ended > rangeTo : true;
  if (startsBefore && endsAfter) {
    return 'spans selected range';
  }
  if (startsBefore) {
    return 'started before range';
  }
  if (endsAfter) {
    return 'continues after range';
  }
  return '';
}

export function formatDateTimeLocalInput(value) {
  const date = toDate(value);
  if (!Number.isFinite(date.getTime())) {
    return '';
  }
  const pad = (part) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function weekStartsOnValue(value, fallback = 'locale') {
  return ['locale', 'monday', 'sunday', 'saturday'].includes(value) ? value : fallback;
}

export function tapSettingsToForm(settings = {}, deviceID = '') {
  return {
    ...defaultTapSettings,
    ...settings,
    deviceID: settings.deviceID || deviceID,
    threshold: byteValue(settings.threshold, defaultTapSettings.threshold),
    limit: byteValue(settings.limit, defaultTapSettings.limit),
    latency: byteValue(settings.latency, defaultTapSettings.latency),
    window: byteValue(settings.window, defaultTapSettings.window),
  };
}

export function tapPresetToForm(preset = {}, deviceID = '') {
  return tapSettingsToForm(preset.settings || {}, deviceID);
}

export function tapFormToSettings(form = {}, deviceID = '') {
  return {
    deviceID,
    threshold: byteValue(form.threshold, defaultTapSettings.threshold),
    limit: byteValue(form.limit, defaultTapSettings.limit),
    latency: byteValue(form.latency, defaultTapSettings.latency),
    window: byteValue(form.window, defaultTapSettings.window),
  };
}

export function tapTuningStatus(state = null, selectedTapSettings = null) {
  if (state?.status === 'restore needed') {
    return 'restore needed';
  }
  if (state?.active && state?.status === 'temporary') {
    return 'temporary';
  }
  if (state?.active) {
    return state.status || 'ready';
  }
  if (selectedTapSettings?.confirmedOnDevice) {
    return 'confirmed on device';
  }
  if (selectedTapSettings) {
    return 'saved locally';
  }
  return 'defaults';
}

export function ledSettingsToForm(settings = {}, deviceID = '') {
  return {
    ...defaultLEDSettings,
    ...settings,
    deviceID: settings.deviceID || deviceID,
    brightnessPercent: rangeValue(settings.brightnessPercent, 1, 100, defaultLEDSettings.brightnessPercent),
    blinkSeconds: rangeValue(settings.blinkSeconds, 5, 60, defaultLEDSettings.blinkSeconds),
  };
}

export function byteValue(value, fallback = 0) {
  return rangeValue(value, 0, 255, fallback);
}

export function rangeValue(value, min, max, fallback = min) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return fallback;
  }
  return Math.max(min, Math.min(max, Math.round(numeric)));
}

function weekStartDay(weekStartsOn, locale) {
  if (weekStartsOn === 'sunday') {
    return 0;
  }
  if (weekStartsOn === 'monday') {
    return 1;
  }
  if (weekStartsOn === 'saturday') {
    return 6;
  }
  try {
    const info = new Intl.Locale(locale).weekInfo;
    if (info?.firstDay === 7) {
      return 0;
    }
    if (Number.isInteger(info?.firstDay)) {
      return info.firstDay;
    }
  } catch {
    return 1;
  }
  return 1;
}

function localMidnight(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function addCalendarDays(date, days) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);
}

function toDate(value) {
  if (value instanceof Date) {
    return value;
  }
  return new Date(value);
}

function defaultLocale() {
  return typeof navigator !== 'undefined' && navigator.language ? navigator.language : 'en-GB';
}

function defaultTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || '';
  } catch {
    return '';
  }
}

export function messageFromError(err) {
  if (!err) {
    return 'Unexpected error.';
  }
  if (typeof err === 'string') {
    const parsed = parseRuntimeError(err);
    return parsed ? messageFromError(parsed) : err;
  }
  if (err.cause?.message) {
    return err.cause.message;
  }
  if (err.message && typeof err.message === 'string') {
    const parsed = parseRuntimeError(err.message);
    if (parsed) {
      return messageFromError(parsed);
    }
    return err.message;
  }
  const encoded = JSON.stringify(err);
  if (!encoded || encoded === '{}') {
    return 'Desktop runtime unavailable. Open TimeFlip Desktop to read device state.';
  }
  return encoded;
}

function parseRuntimeError(value) {
  const trimmed = value.trim();
  if (!trimmed.startsWith('{') || !trimmed.endsWith('}')) {
    return null;
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    return null;
  }
}
