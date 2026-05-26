import assert from 'node:assert/strict';
import test from 'node:test';
import { byteValue, configToSettings, durationToSeconds, ledSettingsToForm, messageFromError, rangeValue, secondsToDuration, tapFormToSettings, tapPresetToForm, tapSettingsToForm, tapTuningStatus } from './timeflip-format.js';

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
  }), {
    communicationTimeoutSeconds: 10,
    commandTimeoutSeconds: 5,
    initialRetrySeconds: 15,
    mediumRetrySeconds: 60,
    longRetrySeconds: 300,
    offlineAfterSeconds: 120,
    offlineAfterFailures: 4,
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
