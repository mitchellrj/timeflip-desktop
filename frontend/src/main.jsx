import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  Bluetooth,
  Clock3,
  History,
  KeyRound,
  Lock,
  Pause,
  Play,
  Plus,
  Plug,
  RefreshCw,
  Save,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Tag,
  Trash2,
  Unlock,
  Unplug,
} from 'lucide-react';
import { Events } from '@wailsio/runtime';
import {
  ConnectDevice,
  DisconnectDevice,
  GetAppState,
  PairDevice,
  SaveDeviceName,
  SaveFacetAssignment,
  SaveLEDSettings,
  SaveSettings,
  SaveTask,
  SaveTapSettings,
  ScanDevices,
  SetLocked,
  SetPaused,
  UnpairDevice,
} from '../bindings/github.com/mitchellrj/timeflip-desktop/internal/app/controller.js';
import { byteValue, configToSettings, defaultLEDSettings, defaultSettings, defaultTapSettings, ledSettingsToForm, messageFromError, rangeValue, secondsToDuration, tapSettingsToForm } from './timeflip-format.js';
import './styles.css';

const emptyState = { config: {}, devices: [], states: [], tapSettings: [], ledSettings: [], tasks: [], sessions: [], facetConfigs: [] };
const defaultTask = { mode: 'task', id: '', label: '', icon: 'tag', color: '#69d2a5', pomodoroLimitMinutes: 25 };
const defaultPair = { deviceID: '', password: '000000', newPassword: '', allowOSPairing: true };
const defaultPassword = { currentPassword: '', newPassword: '', confirmPassword: '' };
const defaultUnpair = { factoryReset: false, allowOSUnpairing: true };
function App() {
  const [state, setState] = useState(emptyState);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState('');
  const [selectedDevice, setSelectedDevice] = useState('');
  const [selectedFacet, setSelectedFacet] = useState(1);
  const [discovered, setDiscovered] = useState([]);
  const [pairForm, setPairForm] = useState(defaultPair);
  const [passwordForm, setPasswordForm] = useState(defaultPassword);
  const [unpairForm, setUnpairForm] = useState(defaultUnpair);
  const [taskForm, setTaskForm] = useState(defaultTask);
  const [facetDirty, setFacetDirty] = useState(false);
  const [settingsForm, setSettingsForm] = useState(defaultSettings);
  const [tapForm, setTapForm] = useState(defaultTapSettings);
  const [ledForm, setLEDForm] = useState(defaultLEDSettings);
  const [deviceNameForm, setDeviceNameForm] = useState('');
  const [workflow, setWorkflow] = useState(null);
  const [now, setNow] = useState(Date.now());
  const refreshVersion = useRef(0);

  async function refresh() {
    const version = refreshVersion.current + 1;
    refreshVersion.current = version;
    const next = await GetAppState();
    if (version !== refreshVersion.current) {
      return next || emptyState;
    }
    const nextState = next || emptyState;
    setState((current) => mergeAppState(current, nextState));
    setSelectedDevice((current) => {
      if (current && nextState.devices?.some((device) => device.id === current)) {
        return current;
      }
      return nextState.devices?.[0]?.id || '';
    });
    setSettingsForm(configToSettings(nextState.config));
    setError('');
    return nextState;
  }

  async function runAction(key, action, success) {
    setBusy(key);
    setError('');
    setNotice('');
    try {
      const result = await action();
      if (success) {
        setNotice(success);
      }
      await refresh();
      return result;
    } catch (err) {
      setError(messageFromError(err));
      return null;
    } finally {
      setBusy('');
    }
  }

  useEffect(() => {
    const refreshFromEvent = () => {
      refresh().catch((err) => setError(messageFromError(err)));
    };
    const refreshFromDeviceState = (event) => {
      const deviceState = event?.data?.state || event?.data;
      if (deviceState?.deviceID) {
        setState((current) => mergeDeviceState(current, deviceState));
      }
    };
    refreshFromEvent();
    const offDeviceState = Events.On('device.state', refreshFromDeviceState);
    const offConnectionState = Events.On('device.connection', refreshFromDeviceState);
    const offHandlers = [
      'shell.refresh',
      'devices.scanned',
      'device.pairing',
      'device.facet.saved',
      'device.profile.saved',
      'device.tap.saved',
      'device.led.saved',
      'device.unpairing',
      'history.imported',
      'tracking.session.started',
      'tracking.session.updated',
      'tracking.session.ended',
    ].map((eventName) => Events.On(eventName, refreshFromEvent));
    const offError = Events.On('device.error', (event) => {
      const message = messageFromError(event?.data || event);
      refresh().catch(() => {}).finally(() => setError(message));
    });
    return () => {
      offDeviceState();
      offConnectionState();
      offHandlers.forEach((off) => off());
      offError();
    };
  }, []);

  useEffect(() => {
    const tick = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(tick);
  }, []);

  const activeState = useMemo(() => state.states?.find((item) => item.deviceID === selectedDevice) || state.states?.[0], [state, selectedDevice]);
  const selectedDeviceView = state.devices?.find((device) => device.id === selectedDevice);
  const currentSession = useMemo(
    () => currentSessionForDevice(state.sessions, selectedDevice || activeState?.deviceID) || state.currentSession,
    [state.sessions, state.currentSession, selectedDevice, activeState?.deviceID],
  );
  const activeFacetConfig = activeState?.currentFacetKnown
    ? state.facetConfigs?.find((item) => item.deviceID === activeState.deviceID && item.facet === activeState.currentFacet)
    : null;
  const currentSessionMatchesState = currentSession
    && (!activeState?.deviceID || currentSession.deviceID === activeState.deviceID)
    && (!activeState?.currentFacetKnown || currentSession.facet === activeState.currentFacet);
  const activeTaskName = taskNameFromActiveState(activeState, activeFacetConfig, currentSessionMatchesState ? currentSession : null);
  const activeFacetLabel = activeState?.currentFacetKnown && !activeState.currentFacetUndefined
    ? `Facet ${activeState.currentFacet}`
    : 'facet unknown';
  const activeModeLabel = [activeState?.locked ? 'locked' : '', activeState?.paused ? 'paused' : ''].filter(Boolean).join(' · ');
  const activeStatusLabel = activeState?.connectionState
    ? `${activeState.connectionState} · ${activeFacetLabel}${activeModeLabel ? ` · ${activeModeLabel}` : ''} · ${activeTaskName}`
    : 'No device connected · task unknown';
  const taskChoices = useMemo(() => dedupeTasks(state.tasks, taskForm.id), [state.tasks, taskForm.id]);
  const selectedFacetConfig = state.facetConfigs?.find((item) => item.deviceID === selectedDevice && item.facet === selectedFacet);
  const selectedTapSettings = state.tapSettings?.find((item) => item.deviceID === selectedDevice);
  const selectedLEDSettings = state.ledSettings?.find((item) => item.deviceID === selectedDevice);
  const selectedFacetSavedLabel = selectedFacetLabel(selectedFacetConfig, selectedFacet);
  const selectedFacetSavedKind = facetKindLabel(selectedFacetConfig);

  useEffect(() => {
    if (selectedDevice) {
      setPairForm((current) => ({ ...current, deviceID: current.deviceID || selectedDevice }));
    }
  }, [selectedDevice]);

  useEffect(() => {
    setDeviceNameForm(selectedDeviceView?.displayName || '');
  }, [selectedDevice, selectedDeviceView?.displayName]);

  useEffect(() => {
    setTaskForm(taskFormFromFacet(selectedFacetConfig, selectedFacet));
    setFacetDirty(false);
  }, [
    selectedDevice,
    selectedFacet,
    selectedFacetConfig?.taskID,
    selectedFacetConfig?.label,
    selectedFacetConfig?.icon,
    selectedFacetConfig?.color,
    selectedFacetConfig?.isPauseAssignment,
    selectedFacetConfig?.isPomodoroAssignment,
    selectedFacetConfig?.pomodoroLimitSeconds,
  ]);

  useEffect(() => {
    setTapForm(tapSettingsToForm(selectedTapSettings, selectedDevice));
  }, [
    selectedDevice,
    selectedTapSettings?.threshold,
    selectedTapSettings?.limit,
    selectedTapSettings?.latency,
    selectedTapSettings?.window,
    selectedTapSettings?.confirmedOnDevice,
  ]);

  useEffect(() => {
    setLEDForm(ledSettingsToForm(selectedLEDSettings, selectedDevice));
  }, [
    selectedDevice,
    selectedLEDSettings?.brightnessPercent,
    selectedLEDSettings?.blinkSeconds,
    selectedLEDSettings?.confirmedOnDevice,
  ]);

  async function scanDevices() {
    const result = await runAction('scan', () => ScanDevices(), 'Scan complete');
    if (result) {
      setDiscovered(result);
      if (result[0]?.id) {
        setPairForm((current) => ({ ...current, deviceID: result[0].id }));
      }
    }
  }

  async function pairDevice(event) {
    event.preventDefault();
    const req = {
      deviceID: pairForm.deviceID.trim(),
      password: pairForm.password.trim(),
      newPassword: pairForm.newPassword.trim(),
      allowOSPairing: pairForm.allowOSPairing,
    };
    if (!req.deviceID) {
      setError('Choose or enter a device ID before pairing.');
      return;
    }
    const result = await runAction('pair', () => PairDevice(req), 'Pairing workflow submitted');
    if (result) {
      setWorkflow(result);
      setPasswordForm(defaultPassword);
      setPairForm((current) => ({ ...current, password: '', newPassword: '' }));
      setSelectedDevice(req.deviceID);
    }
  }

  async function updatePassword(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before updating its password.');
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setError('New password confirmation does not match.');
      return;
    }
    const req = {
      deviceID: selectedDevice,
      password: passwordForm.currentPassword.trim(),
      newPassword: passwordForm.newPassword.trim(),
      allowOSPairing: true,
    };
    const result = await runAction('password', () => PairDevice(req), 'Password update workflow submitted');
    if (result) {
      setWorkflow(result);
      setPasswordForm(defaultPassword);
    }
  }

  async function unpairDevice(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before unpairing.');
      return;
    }
    const result = await runAction(
      'unpair',
      () => UnpairDevice({ deviceID: selectedDevice, factoryReset: unpairForm.factoryReset, allowOSUnpairing: unpairForm.allowOSUnpairing }),
      'Unpairing workflow submitted',
    );
    if (result) {
      setWorkflow(result);
    }
  }

  async function saveTask(event) {
    event.preventDefault();
    const label = taskForm.label.trim();
    if (!label) {
      setError('Task label is required.');
      return;
    }
    const saved = await runAction(
      'task',
      () => SaveTask({ id: taskForm.id, label, icon: taskForm.icon.trim(), color: taskForm.color }),
      'Task saved',
    );
    if (saved) {
      setTaskForm({ id: saved.id, label: saved.label, icon: saved.icon, color: saved.color });
      setFacetDirty(true);
    }
  }

  async function saveFacet(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before saving a facet.');
      return;
    }
    const task = state.tasks?.find((item) => item.id === taskForm.id);
    const existing = selectedFacetConfig || {};
    const label = (taskForm.label || task?.label || existing.label || `Facet ${selectedFacet}`).trim();
    const saved = await runAction(
      'facet',
      () => SaveFacetAssignment({
        deviceID: selectedDevice,
        facet: selectedFacet,
        taskID: taskForm.mode === 'pause' ? '' : taskForm.id,
        label,
        icon: taskForm.icon || task?.icon || existing.icon || 'tag',
        color: taskForm.color || task?.color || existing.color || '#69d2a5',
        isPauseAssignment: taskForm.mode === 'pause',
        isPomodoroAssignment: taskForm.mode === 'pomodoro',
        pomodoroLimitSeconds: taskForm.mode === 'pomodoro' ? minutesToPomodoroSeconds(taskForm.pomodoroLimitMinutes) : 0,
      }),
      'Facet assignment saved',
    );
    if (saved) {
      setFacetDirty(false);
    }
  }

  async function saveSettings(event) {
    event.preventDefault();
    const config = {
      ...state.config,
      communicationTimeout: secondsToDuration(settingsForm.communicationTimeoutSeconds),
      commandTimeout: secondsToDuration(settingsForm.commandTimeoutSeconds),
      reconnectPolicy: {
        ...(state.config?.reconnectPolicy || {}),
        initialRetryInterval: secondsToDuration(settingsForm.initialRetrySeconds),
        mediumRetryInterval: secondsToDuration(settingsForm.mediumRetrySeconds),
        longRetryInterval: secondsToDuration(settingsForm.longRetrySeconds),
        offlineAfterDuration: secondsToDuration(settingsForm.offlineAfterSeconds),
        offlineAfterFailures: Number(settingsForm.offlineAfterFailures || 0),
      },
    };
    await runAction('settings', () => SaveSettings(config), 'Settings saved');
  }

  async function saveTapSettings(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before saving tap settings.');
      return;
    }
    const settings = {
      deviceID: selectedDevice,
      threshold: byteValue(tapForm.threshold, defaultTapSettings.threshold),
      limit: byteValue(tapForm.limit, defaultTapSettings.limit),
      latency: byteValue(tapForm.latency, defaultTapSettings.latency),
      window: byteValue(tapForm.window, defaultTapSettings.window),
    };
    const saved = await runAction('tapSettings', () => SaveTapSettings(settings), 'Tap settings saved');
    if (saved) {
      setTapForm(tapSettingsToForm(saved, selectedDevice));
    }
  }

  function updateTapForm(field, value) {
    setTapForm((current) => ({ ...current, [field]: byteValue(value, defaultTapSettings[field]) }));
  }

  async function saveLEDSettings(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before saving LED settings.');
      return;
    }
    const settings = {
      deviceID: selectedDevice,
      brightnessPercent: rangeValue(ledForm.brightnessPercent, 1, 100, defaultLEDSettings.brightnessPercent),
      blinkSeconds: rangeValue(ledForm.blinkSeconds, 5, 60, defaultLEDSettings.blinkSeconds),
    };
    const saved = await runAction('ledSettings', () => SaveLEDSettings(settings), 'LED settings saved');
    if (saved) {
      setLEDForm(ledSettingsToForm(saved, selectedDevice));
    }
  }

  function updateLEDForm(field, value) {
    const [min, max] = field === 'brightnessPercent' ? [1, 100] : [5, 60];
    setLEDForm((current) => ({ ...current, [field]: rangeValue(value, min, max, defaultLEDSettings[field]) }));
  }

  async function saveDeviceName(event) {
    event.preventDefault();
    if (!selectedDevice) {
      setError('Select a device before saving its name.');
      return;
    }
    const name = deviceNameForm.trim();
    const saved = await runAction('deviceName', () => SaveDeviceName({ deviceID: selectedDevice, name }), 'Device name saved');
    if (saved) {
      setDeviceNameForm(saved.displayName || name);
    }
  }

  function chooseFacetMode(mode) {
    if (mode === 'pause') {
      setTaskForm((current) => ({ ...current, mode, id: '', label: 'Pause', icon: 'pause', color: '#8b95a1' }));
    } else {
      setTaskForm((current) => ({
        ...current,
        mode,
        label: current.mode === 'pause' ? '' : current.label,
        icon: current.mode === 'pause' ? 'tag' : current.icon,
        color: current.mode === 'pause' ? '#69d2a5' : current.color,
        pomodoroLimitMinutes: current.pomodoroLimitMinutes || 25,
      }));
    }
    setFacetDirty(true);
  }

  function chooseTask(taskID) {
    const task = taskChoices.find((item) => item.id === taskID);
    setTaskForm((current) => (
      task
        ? { ...current, id: task.id, label: task.label, icon: task.icon, color: task.color }
        : { ...current, id: '', label: '', icon: 'tag', color: '#69d2a5' }
    ));
    setFacetDirty(true);
  }

  function updateFacetForm(patch) {
    setTaskForm((current) => ({ ...current, ...patch }));
    setFacetDirty(true);
  }

  function resetFacetForm() {
    setTaskForm(taskFormFromFacet(selectedFacetConfig, selectedFacet));
    setFacetDirty(false);
  }

  return (
    <main className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="mark">T</div>
          <div>
            <h1>TimeFlip</h1>
            <p>Local desktop tracking</p>
          </div>
        </div>
        <button className="primary" disabled={busy === 'refresh'} onClick={() => runAction('refresh', refresh, 'State refreshed')}>
          <RefreshCw size={16} /> Refresh
        </button>
        <nav>
          <a href="#dashboard"><Clock3 size={17} /> Dashboard</a>
          <a href="#devices"><Bluetooth size={17} /> Devices</a>
          <a href="#facets"><SlidersHorizontal size={17} /> Facets</a>
          <a href="#history"><History size={17} /> History</a>
          <a href="#settings"><Settings size={17} /> Settings</a>
        </nav>
      </aside>

      <section className="content">
        {error && <div className="notice error">{error}</div>}
        {notice && <div className="notice success">{notice}</div>}

        <section id="dashboard" className="band dashboard">
          <div>
            <span className="eyebrow">Current state</span>
            <h2>{activeTaskName}</h2>
            <p>{activeStatusLabel}</p>
          </div>
          <div className="dashboardActions">
            <button
              className="iconButton"
              disabled={!selectedDevice || busy === 'lock'}
              onClick={() => selectedDevice && runAction('lock', () => SetLocked(selectedDevice, !activeState?.locked), activeState?.locked ? 'Orientation unlocked' : 'Orientation locked')}
              title={activeState?.locked ? 'Unlock orientation' : 'Lock orientation'}
            >
              {activeState?.locked ? <Unlock size={20} /> : <Lock size={20} />}
            </button>
            <button
              className="iconButton"
              disabled={!selectedDevice || busy === 'pause'}
              onClick={() => selectedDevice && runAction('pause', () => SetPaused(selectedDevice, !activeState?.paused), activeState?.paused ? 'Tracking resumed' : 'Tracking paused')}
              title={activeState?.paused ? 'Resume tracking' : 'Pause tracking'}
            >
              {activeState?.paused ? <Play size={20} /> : <Pause size={20} />}
            </button>
          </div>
        </section>

        <section id="devices" className="grid two">
          <Panel title="Devices" icon={<Bluetooth size={18} />}>
            <div className="toolbar">
              <button disabled={busy === 'scan'} onClick={scanDevices}><RefreshCw size={16} /> Scan</button>
              <button disabled={!selectedDevice || busy === 'connect'} onClick={() => runAction('connect', () => ConnectDevice(selectedDevice), 'Connect requested')}><Plug size={16} /> Connect</button>
              <button disabled={!selectedDevice || busy === 'disconnect'} onClick={() => runAction('disconnect', () => DisconnectDevice(selectedDevice), 'Disconnected')}><Unplug size={16} /> Disconnect</button>
            </div>
            <div className="list">
              {state.devices?.map((device) => (
                <button key={device.id} className={`row ${selectedDevice === device.id ? 'selected' : ''}`} onClick={() => setSelectedDevice(device.id)}>
                  <span>{device.displayName || device.id}</span>
                  <small>{device.hasPassword ? 'password stored' : device.pairingState || 'known'}</small>
                </button>
              ))}
              {state.devices?.length === 0 && <p className="empty">No known devices yet.</p>}
            </div>
          </Panel>

          <Panel title="Pairing" icon={<ShieldCheck size={18} />}>
            <form className="form" onSubmit={pairDevice}>
              <label>
                Device
                <select value={pairForm.deviceID} onChange={(event) => setPairForm({ ...pairForm, deviceID: event.target.value })}>
                  <option value="">Choose discovered or enter below</option>
                  {discovered.map((device) => <option key={device.id} value={device.id}>{device.name || device.id}</option>)}
                  {state.devices?.map((device) => <option key={device.id} value={device.id}>{device.displayName || device.id}</option>)}
                </select>
              </label>
              <label>
                Device ID
                <input value={pairForm.deviceID} onChange={(event) => setPairForm({ ...pairForm, deviceID: event.target.value })} placeholder="TimeFlip2 identifier" />
              </label>
              <div className="formGrid">
                <label>
                  Current password
                  <input type="password" value={pairForm.password} onChange={(event) => setPairForm({ ...pairForm, password: event.target.value })} autoComplete="current-password" />
                </label>
                <label>
                  New password
                  <input type="password" value={pairForm.newPassword} onChange={(event) => setPairForm({ ...pairForm, newPassword: event.target.value })} autoComplete="new-password" />
                </label>
              </div>
              <label className="check">
                <input type="checkbox" checked={pairForm.allowOSPairing} onChange={(event) => setPairForm({ ...pairForm, allowOSPairing: event.target.checked })} />
                Allow OS pairing prompts
              </label>
              <button className="primary" disabled={busy === 'pair'}><ShieldCheck size={16} /> Pair</button>
            </form>
          </Panel>
        </section>

        <section className="grid two">
          <Panel title="Status" icon={<Tag size={18} />}>
            <dl className="facts">
              <dt>Device</dt><dd>{selectedDeviceView?.displayName || selectedDevice || 'none'}</dd>
              <dt>Connection</dt><dd>{activeState?.connectionState || 'none'}</dd>
              <dt>Battery</dt><dd>{activeState?.batteryPercent ? `${activeState.batteryPercent}%` : 'unknown'}</dd>
              <dt>Locked</dt><dd>{activeState?.locked ? 'yes' : 'no'}</dd>
              <dt>Paused</dt><dd>{activeState?.paused ? 'yes' : 'no'}</dd>
            </dl>
            <form className="form" onSubmit={saveDeviceName}>
              <label>
                Device name
                <input maxLength={18} value={deviceNameForm} onChange={(event) => setDeviceNameForm(event.target.value)} placeholder="TimeFlip2" />
              </label>
              <button disabled={!selectedDevice || busy === 'deviceName'}><Save size={16} /> Save Device Name</button>
            </form>
            <label className="check inlineAction">
              <input
                type="checkbox"
                checked={Boolean(activeState?.locked)}
                disabled={!selectedDevice || busy === 'lockStatus'}
                onChange={(event) => runAction('lockStatus', () => SetLocked(selectedDevice, event.target.checked), event.target.checked ? 'Orientation locked' : 'Orientation unlocked')}
              />
              Lock orientation
            </label>
            <label className="check inlineAction">
              <input
                type="checkbox"
	                checked={Boolean(activeState?.paused)}
	                disabled={!selectedDevice || busy === 'tapPause'}
	                onChange={(event) => runAction('tapPause', () => SetPaused(selectedDevice, event.target.checked), event.target.checked ? 'Tracking paused' : 'Tracking resumed')}
	              />
	              Pause tracking
	            </label>
	          </Panel>

	          <Panel title="Tap Settings" icon={<SlidersHorizontal size={18} />}>
	            <form className="form" onSubmit={saveTapSettings}>
	              <dl className="facts compact">
	                <dt>Status</dt><dd>{selectedTapSettings?.confirmedOnDevice ? 'confirmed on device' : selectedTapSettings ? 'saved locally' : 'defaults'}</dd>
	              </dl>
	              <div className="formGrid">
	                <ByteField label="Threshold" unit="0-255 register value" value={tapForm.threshold} onChange={(value) => updateTapForm('threshold', value)} />
	                <ByteField label="Limit" unit="0-255 register ticks" value={tapForm.limit} onChange={(value) => updateTapForm('limit', value)} />
	                <ByteField label="Latency" unit="0-255 register ticks" value={tapForm.latency} onChange={(value) => updateTapForm('latency', value)} />
	                <ByteField label="Window" unit="0-255 register ticks" value={tapForm.window} onChange={(value) => updateTapForm('window', value)} />
	              </div>
	              <button className="primary" disabled={!selectedDevice || busy === 'tapSettings'}><Save size={16} /> Save Tap Settings</button>
	            </form>
	          </Panel>

	          <Panel title="LED Settings" icon={<SlidersHorizontal size={18} />}>
	            <form className="form" onSubmit={saveLEDSettings}>
	              <dl className="facts compact">
	                <dt>Status</dt><dd>{selectedLEDSettings?.confirmedOnDevice ? 'confirmed on device' : selectedLEDSettings ? 'saved locally' : 'defaults'}</dd>
	              </dl>
	              <div className="formGrid">
	                <ByteField label="Brightness" min={1} max={100} unit="percent, 1-100" value={ledForm.brightnessPercent} onChange={(value) => updateLEDForm('brightnessPercent', value)} />
	                <ByteField label="Blink" min={5} max={60} unit="seconds, 5-60" value={ledForm.blinkSeconds} onChange={(value) => updateLEDForm('blinkSeconds', value)} />
	              </div>
	              <button className="primary" disabled={!selectedDevice || busy === 'ledSettings'}><Save size={16} /> Save LED Settings</button>
	            </form>
	          </Panel>

	          <Panel title="Password" icon={<KeyRound size={18} />}>
            <form className="form" onSubmit={updatePassword}>
              <p className="helper">Stored passwords are never displayed. Enter the current value only when changing it.</p>
              <label>
                Current password
                <input type="password" value={passwordForm.currentPassword} onChange={(event) => setPasswordForm({ ...passwordForm, currentPassword: event.target.value })} autoComplete="current-password" />
              </label>
              <div className="formGrid">
                <label>
                  New password
                  <input type="password" value={passwordForm.newPassword} onChange={(event) => setPasswordForm({ ...passwordForm, newPassword: event.target.value })} autoComplete="new-password" />
                </label>
                <label>
                  Confirm
                  <input type="password" value={passwordForm.confirmPassword} onChange={(event) => setPasswordForm({ ...passwordForm, confirmPassword: event.target.value })} autoComplete="new-password" />
                </label>
              </div>
              <button disabled={!selectedDevice || busy === 'password'}><KeyRound size={16} /> Update</button>
            </form>
            <form className="form dangerZone" onSubmit={unpairDevice}>
              <div className="formGrid">
                <label className="check"><input type="checkbox" checked={unpairForm.factoryReset} onChange={(event) => setUnpairForm({ ...unpairForm, factoryReset: event.target.checked })} /> Factory reset</label>
                <label className="check"><input type="checkbox" checked={unpairForm.allowOSUnpairing} onChange={(event) => setUnpairForm({ ...unpairForm, allowOSUnpairing: event.target.checked })} /> OS unpairing</label>
              </div>
              <button className="danger" disabled={!selectedDevice || busy === 'unpair'}><Trash2 size={16} /> Unpair</button>
            </form>
          </Panel>
        </section>

        {workflow && <WorkflowStatus workflow={workflow} />}

        <section id="facets" className="band">
          <div className="sectionTitle">
            <h2><SlidersHorizontal size={20} /> Facets</h2>
            <p>Labels and icons stay local. Colours are confirmed on device when connected.</p>
          </div>
          <div className="facetLayout">
            <div className="facetGrid">
              {Array.from({ length: 12 }, (_, index) => {
                const facet = index + 1;
                const cfg = state.facetConfigs?.find((item) => item.facet === facet);
                return (
                  <button className={`facet ${selectedFacet === facet ? 'selected' : ''}`} key={facet} onClick={() => setSelectedFacet(facet)}>
                    <span className="swatch" style={{ background: cfg?.color || '#d8dee9' }} />
                    <strong>{cfg?.label || `Facet ${facet}`}</strong>
                    <small>{facetTileKind(cfg)}</small>
                  </button>
                );
              })}
            </div>
            <form className="form editor" onSubmit={saveFacet}>
              <div className="editorHeader">
                <div>
                  <span className="eyebrow">Editing</span>
                  <h3>Facet {selectedFacet}</h3>
                </div>
                <span className={`statusPill ${facetDirty ? 'dirty' : 'clean'}`}>{facetDirty ? 'Unsaved draft' : 'Saved values'}</span>
              </div>
              <div className="savedFacet">
                <span className="swatch" style={{ background: selectedFacetConfig?.color || '#d8dee9' }} />
                <div>
                  <strong>{selectedFacetSavedLabel}</strong>
                  <small>{selectedFacetSavedKind}{selectedFacetConfig?.assignedOnDevice ? ' · confirmed on device' : ''}</small>
                </div>
              </div>
              <label>
                Facet type
                <select value={taskForm.mode} onChange={(event) => chooseFacetMode(event.target.value)}>
                  <option value="task">Task</option>
                  <option value="pomodoro">Pomodoro</option>
                  <option value="pause">Pause side</option>
                </select>
              </label>
              <label>
                Assign task
                <select value={taskForm.id} disabled={taskForm.mode === 'pause'} onChange={(event) => chooseTask(event.target.value)}>
                  <option value="">New or unassigned task</option>
                  {taskForm.id && !taskChoices.some((task) => task.id === taskForm.id) && (
                    <option value={taskForm.id}>Saved task ({taskForm.id})</option>
                  )}
                  {taskChoices.map((task) => <option key={task.id} value={task.id}>{task.label}</option>)}
                </select>
              </label>
              <div className="formGrid">
                <label>
                  Label
                  <input value={taskForm.label} onChange={(event) => updateFacetForm({ label: event.target.value })} placeholder={selectedFacetConfig?.label || `Facet ${selectedFacet}`} />
                </label>
                <label>
                  Icon
                  <input value={taskForm.icon} onChange={(event) => updateFacetForm({ icon: event.target.value })} placeholder="tag" />
                </label>
              </div>
              <label>
                Colour
                <input type="color" value={taskForm.color} onChange={(event) => updateFacetForm({ color: event.target.value })} />
              </label>
              <NumberField
                label="Pomodoro"
                unit="minutes"
                min={1}
                max={1440}
                value={taskForm.pomodoroLimitMinutes}
                disabled={taskForm.mode !== 'pomodoro'}
                onChange={(value) => updateFacetForm({ pomodoroLimitMinutes: clampNumber(value, 1, 1440) })}
              />
              <div className="toolbar">
                <button type="button" disabled={!facetDirty} onClick={resetFacetForm}>Reset</button>
                <button type="button" disabled={taskForm.mode === 'pause' || busy === 'task'} onClick={saveTask}><Plus size={16} /> Save Task</button>
                <button className="primary" disabled={!selectedDevice || busy === 'facet'}><Save size={16} /> Save Facet {selectedFacet}</button>
              </div>
            </form>
          </div>
        </section>

        <section id="history" className="band">
          <div className="sectionTitle">
            <h2><History size={20} /> Task Sessions</h2>
            <p>Summary reporting comes later; this view starts with reliable sessions.</p>
          </div>
          <div className="sessionList">
            {state.sessions?.slice(0, 12).map((session) => (
              <div className="session" key={session.id}>
                <span className="swatch" style={{ background: session.taskColorSnapshot || '#d8dee9' }} />
                <strong>{sessionTaskName(session, state.tasks)}</strong>
                <dl className="sessionMeta">
                  <dt>Start</dt><dd>{formatDateTime(session.startedAt)}</dd>
                  <dt>Duration</dt><dd>{sessionDurationLabel(session, now)}</dd>
                  <dt>Paused</dt><dd>{sessionPausedLabel(session, now, state.states?.find((item) => item.deviceID === session.deviceID))}</dd>
                  {session.endedAt && <dt>End</dt>}
                  {session.endedAt && <dd>{formatDateTime(session.endedAt)}</dd>}
                </dl>
              </div>
            ))}
            {state.sessions?.length === 0 && <p className="empty">No task sessions recorded yet.</p>}
          </div>
        </section>

        <section id="settings" className="band">
          <div className="sectionTitle">
            <h2><Settings size={20} /> Settings</h2>
            <p>{state.config?.databasePath || 'Local app config'}</p>
          </div>
          <form className="settingsGrid" onSubmit={saveSettings}>
            <NumberField label="Communication timeout" value={settingsForm.communicationTimeoutSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, communicationTimeoutSeconds: value })} />
            <NumberField label="Command timeout" value={settingsForm.commandTimeoutSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, commandTimeoutSeconds: value })} />
            <NumberField label="Initial retry" value={settingsForm.initialRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, initialRetrySeconds: value })} />
            <NumberField label="Medium retry" value={settingsForm.mediumRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, mediumRetrySeconds: value })} />
            <NumberField label="Long retry" value={settingsForm.longRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, longRetrySeconds: value })} />
            <NumberField label="Offline after" value={settingsForm.offlineAfterSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, offlineAfterSeconds: value })} />
            <NumberField label="Failure threshold" value={settingsForm.offlineAfterFailures} onChange={(value) => setSettingsForm({ ...settingsForm, offlineAfterFailures: value })} />
            <button className="primary" disabled={busy === 'settings'}><Save size={16} /> Save Settings</button>
          </form>
        </section>
      </section>
    </main>
  );
}

function Panel({ title, icon, children }) {
  return (
    <section className="panel">
      <h2>{icon}{title}</h2>
      {children}
    </section>
  );
}

function NumberField({ label, value, onChange, unit = '', min = 0, max, disabled = false }) {
  return (
    <label>
      <span className="fieldLabel">
        <span>{label}</span>
        {unit ? <small>{unit}</small> : null}
      </span>
      <input
        type="number"
        min={min}
        max={max}
        value={value}
        disabled={disabled}
        aria-label={unit ? `${label}, ${unit}` : label}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </label>
  );
}

function ByteField({ label, unit, value, onChange, min = 0, max = 255 }) {
  return (
    <label>
      <span className="fieldLabel">
        <span>{label}</span>
        {unit ? <small>{unit}</small> : null}
      </span>
      <input type="number" min={min} max={max} value={value} aria-label={unit ? `${label}, ${unit}` : label} onChange={(event) => onChange(Number(event.target.value))} />
    </label>
  );
}

function WorkflowStatus({ workflow }) {
  return (
    <section className="band workflow">
      <div className="sectionTitle">
        <h2><ShieldCheck size={20} /> Workflow</h2>
        <p>{workflow.deviceID} · {workflow.completed ? 'complete' : workflow.currentStage}</p>
      </div>
      <div className="workflowSteps">
        {workflow.stages?.map((stage) => (
          <div className={`step ${stage.completed ? 'done' : ''}`} key={stage.stage}>
            <strong>{stage.stage}</strong>
            <small>{stage.error || (stage.completed ? 'complete' : stage.manualAction?.description || 'pending')}</small>
          </div>
        ))}
      </div>
      {workflow.manualAction && <p className="helper">{workflow.manualAction.description}</p>}
    </section>
  );
}

function taskFormFromFacet(config, facet) {
  if (config?.isPauseAssignment) {
    return { mode: 'pause', id: '', label: config.label || 'Pause', icon: config.icon || 'pause', color: normaliseColor(config.color, '#8b95a1'), pomodoroLimitMinutes: 25 };
  }
  if (config?.taskID || config?.label || config?.icon || config?.color) {
    return {
      mode: config.isPomodoroAssignment ? 'pomodoro' : 'task',
      id: config.taskID || '',
      label: config.label || '',
      icon: config.icon || 'tag',
      color: normaliseColor(config.color, '#69d2a5'),
      pomodoroLimitMinutes: config.isPomodoroAssignment ? Math.max(1, secondsToMinutes(config.pomodoroLimitSeconds)) : 25,
    };
  }
  return { ...defaultTask, label: '', color: '#69d2a5' };
}

function facetKindLabel(config) {
  if (config?.isPauseAssignment) {
    return 'Pause side';
  }
  if (!config?.taskID) {
    return 'Unassigned';
  }
  if (config.isPomodoroAssignment) {
    return `Pomodoro · ${secondsToMinutes(config.pomodoroLimitSeconds)} min`;
  }
  return 'Task assignment';
}

function facetTileKind(config) {
  if (!config?.taskID && !config?.isPauseAssignment) {
    return 'Unassigned';
  }
  if (config.isPauseAssignment) {
    return 'Pause side';
  }
  if (config.isPomodoroAssignment) {
    return 'Pomodoro';
  }
  return config.icon || 'Task';
}

function selectedFacetLabel(config, facet) {
  if (!config?.taskID && !config?.isPauseAssignment && !config?.label) {
    return `Facet ${facet} is unassigned`;
  }
  return config.label || `Facet ${facet}`;
}

function currentSessionForDevice(sessions = [], deviceID = '') {
  return sessions.find((session) => session.deviceID === deviceID && !session.endedAt) || null;
}

function taskNameFromActiveState(activeState, facetConfig, currentSession) {
  if (!activeState) {
    return 'No active session';
  }
  if (activeState.paused || facetConfig?.isPauseAssignment) {
    return facetConfig?.label || 'Paused';
  }
  if (!activeState.currentFacetKnown || activeState.currentFacetUndefined) {
    return currentSession?.taskLabelSnapshot || 'Task unknown';
  }
  return facetConfig?.label || currentSession?.taskLabelSnapshot || `Facet ${activeState.currentFacet} unassigned`;
}

function sessionTaskName(session, tasks = []) {
  if (session.taskLabelSnapshot) {
    return session.taskLabelSnapshot;
  }
  return tasks.find((task) => task.id === session.taskID)?.label || 'Unnamed task';
}

function sessionDurationLabel(session, now = Date.now()) {
  return formatDurationSeconds(sessionDurationSeconds(session, now));
}

function sessionPausedLabel(session, now = Date.now(), deviceState = null) {
  let seconds = Number(session.pausedSeconds || 0);
  if (session.pauseStartedAt && !session.endedAt && deviceState?.paused) {
    seconds += secondsBetween(session.pauseStartedAt, now);
  }
  return formatDurationSeconds(seconds);
}

function sessionDurationSeconds(session, now = Date.now()) {
  if (session.endedAt) {
    const stored = Number(session.durationSeconds || 0);
    return stored > 0 ? stored : secondsBetween(session.startedAt, session.endedAt);
  }
  return secondsBetween(session.startedAt, now);
}

function mergeDeviceState(current, deviceState) {
  const states = current.states || [];
  const found = states.some((state) => state.deviceID === deviceState.deviceID);
  return {
    ...current,
    states: found
      ? states.map((state) => (state.deviceID === deviceState.deviceID ? newerDeviceState(state, deviceState) : state))
      : [...states, deviceState],
  };
}

function mergeAppState(current, next) {
  return {
    ...next,
    states: mergeDeviceStates(current.states, next.states),
  };
}

function mergeDeviceStates(currentStates = [], nextStates = []) {
  const byDevice = new Map();
  for (const state of currentStates) {
    if (state?.deviceID) {
      byDevice.set(state.deviceID, state);
    }
  }
  for (const state of nextStates) {
    if (!state?.deviceID) {
      continue;
    }
    byDevice.set(state.deviceID, newerDeviceState(byDevice.get(state.deviceID), state));
  }
  return Array.from(byDevice.values());
}

function newerDeviceState(current, next) {
  if (!current) {
    return next;
  }
  if (deviceStateMillis(current) > deviceStateMillis(next)) {
    return current;
  }
  return { ...current, ...next };
}

function deviceStateMillis(state) {
  const value = new Date(state?.updatedAt || 0).getTime();
  return Number.isFinite(value) ? value : 0;
}

function secondsBetween(start, end) {
  const started = new Date(start).getTime();
  const ended = typeof end === 'number' ? end : new Date(end).getTime();
  if (!Number.isFinite(started) || !Number.isFinite(ended) || ended <= started) {
    return 0;
  }
  return Math.floor((ended - started) / 1000);
}

function formatDurationSeconds(seconds) {
  const safeSeconds = Math.max(0, Number(seconds || 0));
  if (safeSeconds < 45) {
    return `${Math.floor(safeSeconds)} sec`;
  }
  if (safeSeconds < 90) {
    return '1 min';
  }
  const minutes = Math.floor(safeSeconds / 60);
  const remainingSeconds = Math.floor(safeSeconds % 60);
  if (minutes < 60) {
    return remainingSeconds > 0 ? `${minutes} min ${remainingSeconds} sec` : `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours < 24) {
    return remainingMinutes > 0 ? `${hours} ${unit(hours, 'hr')} ${remainingMinutes} min` : `${hours} ${unit(hours, 'hr')}`;
  }
  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return remainingHours > 0 ? `${days} ${unit(days, 'day')} ${remainingHours} ${unit(remainingHours, 'hr')}` : `${days} ${unit(days, 'day')}`;
}

function unit(value, singular) {
  return value === 1 ? singular : `${singular}s`;
}

function formatDateTime(value) {
  if (!value) {
    return 'Not recorded';
  }
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) {
    return 'Not recorded';
  }
  return date.toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' });
}

function dedupeTasks(tasks = [], selectedTaskID = '') {
  const byLabel = new Map();
  for (const task of tasks) {
    const key = normaliseTaskLabel(task.label);
    if (!key) {
      continue;
    }
    const existing = byLabel.get(key);
    if (!existing || task.id === selectedTaskID) {
      byLabel.set(key, task);
    }
  }
  return Array.from(byLabel.values());
}

function normaliseTaskLabel(label) {
  return String(label || '').trim().replace(/\s+/g, ' ').toLocaleLowerCase();
}

function normaliseColor(color, fallback) {
  return /^#[0-9a-fA-F]{6}$/.test(color || '') ? color : fallback;
}

function secondsToMinutes(seconds) {
  return Math.round(Number(seconds || 0) / 60);
}

function minutesToPomodoroSeconds(minutes) {
  return clampNumber(minutes, 1, 1440) * 60;
}

function clampNumber(value, min, max) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return min;
  }
  return Math.min(max, Math.max(min, Math.round(parsed)));
}

createRoot(document.getElementById('root')).render(<App />);
