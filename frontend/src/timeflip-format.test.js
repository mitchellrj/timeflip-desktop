import assert from 'node:assert/strict';
import test from 'node:test';
import { byteValue, compactDuration, configToSettings, durationToSeconds, formatDateTimeLocalInput, historyOverlapLabel, ledSettingsToForm, messageFromError, rangeValue, reportPeriodForPreset, secondsToDuration, summaryBarPercent, tapFormToSettings, tapPresetToForm, tapSettingsToForm, tapTuningStatus, toControllerPeriodRequest, validateCustomPeriod } from './timeflip-format.js';

test('converts Go duration nanoseconds to seconds', () => {
  assert.equal(durationToSeconds(15_000_000_000, 1), 15);
  assert.equal(durationToSeconds(0, 7), 7);
  assert.equal(durationToSeconds(undefined, 9), 9);
});

test('converts settings seconds to Go duration nanoseconds', () => {
  assert.equal(secondsToDuration(5), 5_000_000_000);
  assert.equal(secondsToDuration(-5), 0);
});

test('maps app config to editable settings', () => {
  assert.deepEqual(configToSettings({
    communicationTimeout: 10_000_000_000,
    commandTimeout: 5_000_000_000,
    reconnectPolicy: {
      initialRetryInterval: 15_000_000_000,
      mediumRetryInterval: 60_000_000_000,
      longRetryInterval: 300_000_000_000,
      offlineAfterDuration: 120_000_000_000,
      offlineAfterFailures: 4,
    },
    weekStartsOn: 'monday',
  }), {
    communicationTimeoutSeconds: 10,
    commandTimeoutSeconds: 5,
    initialRetrySeconds: 15,
    mediumRetrySeconds: 60,
    longRetrySeconds: 300,
    offlineAfterSeconds: 120,
    offlineAfterFailures: 4,
    weekStartsOn: 'monday',
  });
});

test('maps tap settings to editable byte fields', () => {
  assert.deepEqual(tapSettingsToForm({ deviceID: 'd1', threshold: 21, limit: 9, latency: 4, window: 31, confirmedOnDevice: true }), {
    deviceID: 'd1',
    threshold: 21,
    limit: 9,
    latency: 4,
    window: 31,
    confirmedOnDevice: true,
  });
  assert.equal(byteValue(300, 20), 255);
  assert.equal(byteValue(-1, 20), 0);
  assert.equal(byteValue('nope', 20), 20);
});

test('maps tap presets to selected device forms', () => {
  assert.deepEqual(tapPresetToForm({
    id: 'sensitive',
    settings: { threshold: 14, limit: 8, latency: 3, window: 24 },
  }, 'd2'), {
    deviceID: 'd2',
    threshold: 14,
    limit: 8,
    latency: 3,
    window: 24,
    confirmedOnDevice: false,
  });
});

test('converts tap form to clamped settings', () => {
  assert.deepEqual(tapFormToSettings({ threshold: 300, limit: -4, latency: 'nope', window: 33 }, 'd3'), {
    deviceID: 'd3',
    threshold: 255,
    limit: 0,
    latency: 5,
    window: 33,
  });
});

test('labels tap tuning status', () => {
  assert.equal(tapTuningStatus({ active: true, status: 'temporary' }, null), 'temporary');
  assert.equal(tapTuningStatus({ active: true, status: 'restore needed' }, null), 'restore needed');
  assert.equal(tapTuningStatus(null, { confirmedOnDevice: true }), 'confirmed on device');
  assert.equal(tapTuningStatus(null, { confirmedOnDevice: false }), 'saved locally');
  assert.equal(tapTuningStatus(null, null), 'defaults');
});

test('maps LED settings to editable fields', () => {
  assert.deepEqual(ledSettingsToForm({ deviceID: 'd1', brightnessPercent: 60, blinkSeconds: 15, confirmedOnDevice: true }), {
    deviceID: 'd1',
    brightnessPercent: 60,
    blinkSeconds: 15,
    confirmedOnDevice: true,
  });
  assert.equal(rangeValue(0, 1, 100, 50), 1);
  assert.equal(rangeValue(120, 1, 100, 50), 100);
  assert.equal(rangeValue('nope', 1, 100, 50), 50);
});

test('normalises frontend error messages', () => {
  assert.equal(messageFromError('plain'), 'plain');
  assert.equal(messageFromError(new Error('boom')), 'boom');
  assert.equal(messageFromError({}), 'Desktop runtime unavailable. Open TimeFlip Desktop to read device state.');
  assert.equal(messageFromError({
    message: 'device_timeout: Device operation failed.',
    cause: {
      code: 'bluetooth_unavailable',
      message: 'Bluetooth is still starting. Try connecting again in a moment.',
      diagnostic: 'connect device d1: macos_adapter: already calling Enable function',
    },
    kind: 'RuntimeError',
  }), 'Bluetooth is still starting. Try connecting again in a moment.');
  assert.equal(messageFromError(JSON.stringify({
    message: 'device_timeout: Device operation failed.',
    cause: {
      code: 'bluetooth_unavailable',
      message: 'Bluetooth is unavailable or turned off.',
      diagnostic: 'macos_adapter: timeout enabling CentralManager',
    },
    kind: 'RuntimeError',
  })), 'Bluetooth is unavailable or turned off.');
});

test('builds locale-aware report preset periods', () => {
  const now = new Date(2026, 4, 27, 15, 30);
  const today = reportPeriodForPreset('today', now, { locale: 'en-GB', weekStartsOn: 'locale' });
  assert.equal(formatDateTimeLocalInput(today.from), '2026-05-27T00:00');
  assert.equal(formatDateTimeLocalInput(today.to), '2026-05-28T00:00');

  const yesterday = reportPeriodForPreset('yesterday', now, { locale: 'en-GB' });
  assert.equal(formatDateTimeLocalInput(yesterday.from), '2026-05-26T00:00');
  assert.equal(formatDateTimeLocalInput(yesterday.to), '2026-05-27T00:00');

  const week = reportPeriodForPreset('this-week', now, { locale: 'en-GB', weekStartsOn: 'monday' });
  assert.equal(formatDateTimeLocalInput(week.from), '2026-05-25T00:00');
  assert.equal(formatDateTimeLocalInput(week.to), '2026-06-01T00:00');

  const month = reportPeriodForPreset('this-month', now, { locale: 'en-GB' });
  assert.equal(formatDateTimeLocalInput(month.from), '2026-05-01T00:00');
  assert.equal(formatDateTimeLocalInput(month.to), '2026-06-01T00:00');
});

test('uses calendar days for preset boundaries across daylight saving changes', () => {
  const now = new Date(2026, 2, 29, 12, 0);
  const period = reportPeriodForPreset('today', now, { locale: 'en-GB' });
  assert.equal(formatDateTimeLocalInput(period.from), '2026-03-29T00:00');
  assert.equal(formatDateTimeLocalInput(period.to), '2026-03-30T00:00');
});

test('validates custom report periods', () => {
  const valid = validateCustomPeriod('2026-05-27T09:00', '2026-05-27T10:00', { locale: 'en-GB', weekStartsOn: 'monday' });
  assert.equal(valid.valid, true);
  assert.equal(valid.period.weekStartsOn, 'monday');
  assert.equal(validateCustomPeriod('', '2026-05-27T10:00').error, 'Start and end are required.');
  assert.equal(validateCustomPeriod('2026-05-27T10:00', '2026-05-27T10:00').error, 'Start must be before end.');
  assert.equal(validateCustomPeriod('2026-05-27T11:00', '2026-05-27T10:00').error, 'Start must be before end.');
});

test('maps report periods to controller request dates', () => {
  const from = new Date(2026, 4, 27, 9, 0);
  const to = new Date(2026, 4, 27, 10, 0);
  const now = new Date(2026, 4, 27, 9, 30);
  const request = toControllerPeriodRequest({ from, to }, now);
  assert.equal(request.from, from);
  assert.equal(request.to, to);
  assert.equal(request.now, now);
});

test('formats compact durations', () => {
  assert.equal(compactDuration(45), '45s');
  assert.equal(compactDuration(90), '1m');
  assert.equal(compactDuration(3660), '1h 1m');
  assert.equal(compactDuration(172800), '2d');
  assert.equal(compactDuration(183600), '2d 3h');
});

test('calculates summary bar percentages', () => {
  const rows = [{ activeSeconds: 100 }, { activeSeconds: 40 }, { activeSeconds: 0 }];
  assert.equal(summaryBarPercent(rows[0], rows), 100);
  assert.equal(summaryBarPercent(rows[1], rows), 40);
  assert.equal(summaryBarPercent(rows[2], rows), 0);
});

test('labels history rows that overlap selected range boundaries', () => {
  const from = new Date(2026, 4, 27, 10, 0);
  const to = new Date(2026, 4, 27, 11, 0);
  assert.equal(historyOverlapLabel({ startedAt: new Date(2026, 4, 27, 9, 0), endedAt: new Date(2026, 4, 27, 10, 30) }, from, to), 'started before range');
  assert.equal(historyOverlapLabel({ startedAt: new Date(2026, 4, 27, 10, 30), endedAt: new Date(2026, 4, 27, 12, 0) }, from, to), 'continues after range');
  assert.equal(historyOverlapLabel({ startedAt: new Date(2026, 4, 27, 9, 0), endedAt: new Date(2026, 4, 27, 12, 0) }, from, to), 'spans selected range');
  assert.equal(historyOverlapLabel({ startedAt: new Date(2026, 4, 27, 10, 15), endedAt: new Date(2026, 4, 27, 10, 45) }, from, to), '');
});
