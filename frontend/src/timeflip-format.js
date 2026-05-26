export const defaultSettings = {
  communicationTimeoutSeconds: 10,
  commandTimeoutSeconds: 5,
  initialRetrySeconds: 15,
  mediumRetrySeconds: 60,
  longRetrySeconds: 300,
  offlineAfterSeconds: 120,
  offlineAfterFailures: 3,
};

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
