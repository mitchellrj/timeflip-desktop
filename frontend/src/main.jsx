import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  Activity,
  BatteryFull,
  Bluetooth,
  BookMarked,
  BriefcaseBusiness,
  Bug,
  CalendarCheck,
  Camera,
  ChartNoAxesCombined,
  ChartPie,
  Check,
  ChevronDown,
  CircleDollarSign,
  Clock3,
  Coffee,
  Code2,
  CodeXml,
  Cog,
  Dumbbell,
  Facebook,
  File,
  FileSearch,
  FileText,
  Gamepad2,
  Globe2,
  GraduationCap,
  Handshake,
  HardHat,
  Headset,
  History,
  Home,
  Instagram,
  Key,
  KeyRound,
  Layers,
  Lightbulb,
  List,
  Lock,
  MailOpen,
  Megaphone,
  MessagesSquare,
  MicVocal,
  Music2,
  Palette,
  Pause,
  PenTool,
  Pencil,
  Phone,
  Play,
  Plus,
  Plug,
  Puzzle,
  RefreshCw,
  Repeat2,
  RotateCcw,
  Save,
  Search,
  Settings,
  ShieldCheck,
  ShoppingCart,
  SlidersHorizontal,
  Tag,
  Trash2,
  Truck,
  Tv,
  Twitter,
  Unlock,
  Unplug,
  UsersRound,
  X,
  Youtube,
  Zap,
} from 'lucide-react';
import { Events } from '@wailsio/runtime';
import {
  BeginTapTuning,
  CancelTapTuning,
  ClearFacetConfiguration,
  ConfirmTapTuningSettings,
  ConnectDevice,
  DisconnectDevice,
  GetAppState,
  ListTapTuningPresets,
  PairDevice,
  PreviewTapTuningSettings,
  ResetFacetConfiguration,
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
import { byteValue, configToSettings, defaultLEDSettings, defaultSettings, defaultTapSettings, ledSettingsToForm, messageFromError, rangeValue, secondsToDuration, tapFormToSettings, tapPresetToForm, tapSettingsToForm, tapTuningStatus } from './timeflip-format.js';
import './styles.css';

const emptyState = { config: {}, devices: [], states: [], tapSettings: [], tapTuningStates: [], ledSettings: [], tasks: [], sessions: [], facetConfigs: [] };
const defaultTask = { mode: 'task', id: '', label: '', icon: 'hard-hat', color: '#69d2a5', pomodoroLimitMinutes: 25 };
const defaultPair = { deviceID: '', password: '000000', newPassword: '', allowOSPairing: true };
const defaultPassword = { currentPassword: '', newPassword: '', confirmPassword: '' };
const defaultUnpair = { factoryReset: false, allowOSUnpairing: true };
const stickerIconOptions = [
  { value: 'hard-hat', label: 'Project', icon: HardHat },
  { value: 'briefcase', label: 'Office', icon: BriefcaseBusiness },
  { value: 'handshake', label: 'Client', icon: Handshake },
  { value: 'zap', label: 'Urgent', icon: Zap },
  { value: 'mail-open', label: 'Emails', icon: MailOpen },
  { value: 'phone', label: 'Calls', icon: Phone },
  { value: 'lightbulb', label: 'Brainstorm', icon: Lightbulb },
  { value: 'users-round', label: 'Meeting', icon: UsersRound },
  { value: 'chart-pie', label: 'Report', icon: ChartPie },
  { value: 'repeat-2', label: 'Agile', icon: Repeat2 },
  { value: 'circle-dollar-sign', label: 'Finances', icon: CircleDollarSign },
  { value: 'truck', label: 'Logistics', icon: Truck },
  { value: 'key', label: 'Admin', icon: Key },
  { value: 'code-xml', label: 'Code', icon: CodeXml },
  { value: 'bug', label: 'Debug', icon: Bug },
  { value: 'cog', label: 'Test', icon: Cog },
  { value: 'ux', label: 'UX', icon: UXIcon },
  { value: 'palette', label: 'Design', icon: Palette },
  { value: 'pen-tool', label: 'Write', icon: PenTool },
  { value: 'pencil', label: 'Edit', icon: Pencil },
  { value: 'shopping-cart', label: 'Sales', icon: ShoppingCart },
  { value: 'megaphone', label: 'Marketing', icon: Megaphone },
  { value: 'puzzle', label: 'Consult', icon: Puzzle },
  { value: 'mic-vocal', label: 'Media', icon: MicVocal },
  { value: 'graduation-cap', label: 'Study', icon: GraduationCap },
  { value: 'book-marked', label: 'Read', icon: BookMarked },
  { value: 'dumbbell', label: 'Fitness', icon: Dumbbell },
  { value: 'camera', label: 'Camera', icon: Camera },
  { value: 'gamepad-2', label: 'Games', icon: Gamepad2 },
  { value: 'music-2', label: 'Music', icon: Music2 },
  { value: 'tv', label: 'TV', icon: Tv },
  { value: 'coffee', label: 'Break', icon: Coffee },
  { value: 'facebook', label: 'Facebook', icon: Facebook },
  { value: 'instagram', label: 'Instagram', icon: Instagram },
  { value: 'youtube', label: 'Youtube', icon: Youtube },
  { value: 'twitter', label: 'Twitter', icon: Twitter },
  { value: 'headset', label: 'Support', icon: Headset },
  { value: 'quotation', label: 'Quotation', icon: QuotationIcon },
  { value: 'file', label: 'Document', icon: File },
  { value: 'chart-no-axes-combined', label: 'Presentation', icon: ChartNoAxesCombined },
  { value: 'globe-2', label: 'Web', icon: Globe2 },
  { value: 'messages-square', label: 'Chat', icon: MessagesSquare },
];
const legacyIconAliases = {
  tag: Tag,
  list: List,
  code: Code2,
  file: File,
  'file-search': FileSearch,
  search: Search,
  calendar: CalendarCheck,
  home: Home,
  pause: Pause,
  layers: Layers,
};
const customIconOptions = [
  ...Object.entries(legacyIconAliases)
    .filter(([value]) => !stickerIconOptions.some((option) => option.value === value))
    .map(([value, icon]) => ({ value, label: humaniseIconName(value), icon })),
].sort((left, right) => left.label.localeCompare(right.label));
const facetColorOptions = ['#69d2a5', '#b83220', '#7ccba2', '#8d2fb7', '#f4b79a', '#fff7a8', '#48a4cf', '#d96f93', '#153f8a', '#f47422', '#fff36a', '#9b7cf2'];
function App() {
  const [state, setState] = useState(emptyState);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState('');
  const [currentPage, setCurrentPage] = useState('track');
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
  const [tapPresets, setTapPresets] = useState([]);
  const [ledForm, setLEDForm] = useState(defaultLEDSettings);
  const [deviceNameForm, setDeviceNameForm] = useState('');
  const [workflow, setWorkflow] = useState(null);
  const [pendingClearFacet, setPendingClearFacet] = useState(null);
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
      return nextState.devices?.find(isPairedDevice)?.id || nextState.devices?.[0]?.id || '';
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
    const refreshFromTapTuningState = (event) => {
      const tapTuningState = event?.data?.state || event?.data;
      if (tapTuningState?.deviceID) {
        setState((current) => mergeTapTuningState(current, tapTuningState));
      }
    };
    const refreshFromTapTuningObservation = (event) => {
      const observation = event?.data?.observation || event?.data;
      if (observation?.deviceID) {
        setState((current) => mergeTapTuningObservation(current, observation));
      }
    };
    refreshFromEvent();
    const offDeviceState = Events.On('device.state', refreshFromDeviceState);
    const offConnectionState = Events.On('device.connection', refreshFromDeviceState);
    const offTapTuningState = Events.On('device.tap.tuning.state', refreshFromTapTuningState);
    const offTapTuningDetected = Events.On('device.tap.tuning.detected', refreshFromTapTuningObservation);
    const offHandlers = [
      'shell.refresh',
      'devices.scanned',
      'device.pairing',
      'device.facet.saved',
      'device.facet.cleared',
      'device.facets.reset',
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
      offTapTuningState();
      offTapTuningDetected();
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
  const activeTaskIcon = activeFacetConfig?.icon || currentSession?.taskIconSnapshot || (activeState?.paused ? 'pause' : defaultTask.icon);
  const activeTaskColor = activeFacetConfig?.color || currentSession?.taskColorSnapshot || '#d8dee9';
  const activeFacetLabel = activeState?.currentFacetKnown && !activeState.currentFacetUndefined
    ? `Facet ${activeState.currentFacet}`
    : 'facet unknown';
  const activeModeLabel = [activeState?.locked ? 'locked' : '', activeState?.paused ? 'paused' : ''].filter(Boolean).join(' · ');
  const activeStatusLabel = activeState?.connectionState
    ? `${activeState.connectionState} · ${activeFacetLabel}${activeModeLabel ? ` · ${activeModeLabel}` : ''} · ${activeTaskName}`
    : 'No device connected · task unknown';
  const taskChoices = useMemo(() => dedupeTasks(state.tasks, taskForm.id), [state.tasks, taskForm.id]);
  const selectedFacetConfig = state.facetConfigs?.find((item) => item.deviceID === selectedDevice && item.facet === selectedFacet);
  const liveFacet = activeState?.deviceID === selectedDevice && activeState?.currentFacetKnown && !activeState?.currentFacetUndefined
    ? activeState.currentFacet
    : 0;
  const selectedTapSettings = state.tapSettings?.find((item) => item.deviceID === selectedDevice);
  const selectedTapTuning = state.tapTuningStates?.find((item) => item.deviceID === selectedDevice);
  const selectedTapStatus = tapTuningStatus(selectedTapTuning, selectedTapSettings);
  const selectedLEDSettings = state.ledSettings?.find((item) => item.deviceID === selectedDevice);
  const selectedDeviceConnected = activeState?.deviceID === selectedDevice && activeState?.connectionState === 'connected';
  const selectedFacetSavedLabel = selectedFacetLabel(selectedFacetConfig, selectedFacet);
  const selectedFacetSavedKind = facetKindLabel(selectedFacetConfig);
  const clearFacetKey = `${selectedDevice}:${selectedFacet}`;
  const clearFacetArmed = pendingClearFacet === clearFacetKey;
  const pairedDevices = useMemo(() => state.devices?.filter(isPairedDevice) || [], [state.devices]);
  const hasPairedDevice = pairedDevices.length > 0;
  const visiblePage = hasPairedDevice ? currentPage : 'device';
  const hasKnownDevice = Boolean(state.devices?.length);
  const hasConfiguredFacets = Boolean(state.facetConfigs?.some((item) => item.deviceID === selectedDevice && (item.taskID || item.isPauseAssignment)));
  const hasSessions = Boolean(state.sessions?.length);
  const currentSessionLabel = currentSession ? sessionDurationLabel(currentSession, now) : 'No running session';
  const currentPausedLabel = currentSession ? sessionPausedLabel(currentSession, now, activeState) : '0 sec';
  const taskFormUsesCustomIcon = isCustomTaskIcon(taskForm.icon);

  useEffect(() => {
    if (selectedDevice) {
      setPairForm((current) => ({ ...current, deviceID: current.deviceID || selectedDevice }));
    }
  }, [selectedDevice]);

  useEffect(() => {
    setPendingClearFacet(null);
  }, [selectedDevice, selectedFacet]);

  useEffect(() => {
    if (!pendingClearFacet) {
      return undefined;
    }
    const timeout = window.setTimeout(() => {
      setPendingClearFacet((current) => (current === pendingClearFacet ? null : current));
    }, 10_000);
    return () => window.clearTimeout(timeout);
  }, [pendingClearFacet]);

  useEffect(() => {
    setDeviceNameForm(selectedDeviceView?.displayName || '');
  }, [selectedDevice, selectedDeviceView?.displayName]);

  useEffect(() => {
    if (!selectedDevice) {
      setTapPresets([]);
      return;
    }
    Promise.resolve(ListTapTuningPresets(selectedDevice))
      .then((presets) => setTapPresets(presets || []))
      .catch((err) => setError(messageFromError(err)));
  }, [selectedDevice]);

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
    const source = selectedTapTuning?.active ? selectedTapTuning.draftSettings : selectedTapSettings;
    setTapForm(tapSettingsToForm(source, selectedDevice));
  }, [
    selectedDevice,
    selectedTapTuning?.active,
    selectedTapTuning?.draftSettings?.threshold,
    selectedTapTuning?.draftSettings?.limit,
    selectedTapTuning?.draftSettings?.latency,
    selectedTapTuning?.draftSettings?.window,
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
      setCurrentPage(result.completed ? 'facets' : 'device');
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
        icon: taskForm.icon || task?.icon || existing.icon || defaultTask.icon,
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

  async function resetAllFacets() {
    if (!selectedDevice) {
      setError('Select a device before resetting facets.');
      return;
    }
    if (!window.confirm('Reset all facet assignments for this device? Existing tasks and session history will stay intact.')) {
      return;
    }
    const reset = await runAction(
      'resetFacets',
      () => ResetFacetConfiguration(selectedDevice),
      'Facet configuration reset',
    );
    if (reset) {
      setState((current) => replaceFacetConfigsForDevice(current, selectedDevice, reset));
      setSelectedFacet(1);
      setFacetDirty(false);
      setTaskForm(taskFormFromFacet(null, 1));
    }
  }

  async function clearFacet() {
    if (!selectedDevice) {
      setError('Select a device before clearing a facet.');
      return;
    }
    if (!clearFacetArmed) {
      setPendingClearFacet(clearFacetKey);
      setError('');
      setNotice(`Click Clear Facet again to clear Facet ${selectedFacet}. Existing tasks and session history will stay intact.`);
      return;
    }
    setPendingClearFacet(null);
    const cleared = await runAction(
      'clearFacet',
      () => ClearFacetConfiguration(selectedDevice, selectedFacet),
      `Facet ${selectedFacet} cleared`,
    );
    if (cleared) {
      setState((current) => upsertFacetConfig(current, { ...cleared, deviceID: selectedDevice, facet: selectedFacet }));
      setFacetDirty(false);
      setTaskForm(taskFormFromFacet(null, selectedFacet));
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
    const settings = tapFormToSettings(tapForm, selectedDevice);
    const action = selectedTapTuning?.active
      ? () => ConfirmTapTuningSettings(settings)
      : () => SaveTapSettings(settings);
    const saved = await runAction('tapSettings', action, 'Tap settings saved');
    if (saved) {
      setTapForm(tapSettingsToForm(saved, selectedDevice));
    }
  }

  async function beginTapTuning() {
    if (!selectedDevice) {
      setError('Select a device before tuning tap settings.');
      return;
    }
    const result = await runAction('tapTuningStart', () => BeginTapTuning(selectedDevice), 'Tap tuning started');
    if (result) {
      setState((current) => mergeTapTuningState(current, result));
      setTapForm(tapSettingsToForm(result.draftSettings, selectedDevice));
    }
  }

  async function previewTapTuning() {
    if (!selectedDevice) {
      setError('Select a device before applying tap settings.');
      return;
    }
    const settings = tapFormToSettings(tapForm, selectedDevice);
    const result = await runAction('tapTuningPreview', () => PreviewTapTuningSettings(settings), 'Temporary tap settings applied');
    if (result) {
      setState((current) => mergeTapTuningState(current, result));
      setTapForm(tapSettingsToForm(result.draftSettings, selectedDevice));
    }
  }

  async function cancelTapTuning() {
    if (!selectedDevice) {
      setError('Select a device before cancelling tap tuning.');
      return;
    }
    const result = await runAction('tapTuningCancel', () => CancelTapTuning(selectedDevice), 'Tap tuning cancelled');
    if (result) {
      setState((current) => mergeTapTuningState(current, result));
      setTapForm(tapSettingsToForm(selectedTapSettings, selectedDevice));
    }
  }

  function applyTapPreset(preset) {
    setTapForm(tapPresetToForm(preset, selectedDevice));
  }

  function resetTapForm() {
    const source = selectedTapTuning?.active ? selectedTapTuning.originalSettings : selectedTapSettings;
    setTapForm(tapSettingsToForm(source, selectedDevice));
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
        icon: current.mode === 'pause' ? defaultTask.icon : current.icon,
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
        : { ...current, id: '', label: '', icon: defaultTask.icon, color: '#69d2a5' }
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
          <div className="mark" aria-hidden="true"><SandtimerLogo /></div>
          <div>
            <h1>TimeFlip</h1>
            <p>Local desktop tracking</p>
          </div>
        </div>
        <button className="primary" disabled={busy === 'refresh'} onClick={() => runAction('refresh', refresh, 'State refreshed')}>
          <RefreshCw size={16} /> Refresh
        </button>
        <nav aria-label="Primary">
          {hasPairedDevice ? (
            <>
              <NavButton page="track" currentPage={visiblePage} onNavigate={setCurrentPage} icon={<Clock3 size={17} />}>Track</NavButton>
              <NavButton page="facets" currentPage={visiblePage} onNavigate={setCurrentPage} icon={<SlidersHorizontal size={17} />}>Facets</NavButton>
              <NavButton page="config" currentPage={visiblePage} onNavigate={setCurrentPage} icon={<Settings size={17} />}>Device config</NavButton>
              <NavButton page="device" currentPage={visiblePage} onNavigate={setCurrentPage} icon={<Bluetooth size={17} />}>Device management</NavButton>
            </>
          ) : (
            <NavButton page="device" currentPage={visiblePage} onNavigate={setCurrentPage} icon={<Bluetooth size={17} />}>Pair device</NavButton>
          )}
        </nav>
        <a className="navButton sidebarBugLink" href="https://github.com/mitchellrj/timeflip-desktop/issues/new" target="_blank" rel="noreferrer">
          <Bug size={17} /> Report a bug
        </a>
      </aside>

      <section className="content">
        {error && (
          <DismissibleNotice kind="error" onDismiss={() => setError('')}>
            {error}
          </DismissibleNotice>
        )}
        {notice && (
          <DismissibleNotice kind="success" onDismiss={() => setNotice('')}>
            {notice}
          </DismissibleNotice>
        )}

        {visiblePage === 'track' && (
        <section id="dashboard" className="band dashboard">
          <div className="currentState">
            <IconBadge name={activeTaskIcon} color={activeTaskColor} size={24} />
            <div>
              <span className="eyebrow">Now tracking</span>
              <h2>{activeTaskName}</h2>
              <p>{activeStatusLabel}</p>
            </div>
            <div className="statusChips" aria-label="Tracking status">
              <span className={`chip ${selectedDeviceConnected ? 'good' : 'muted'}`}>
                <Activity size={14} /> {selectedDeviceConnected ? 'Connected' : 'Not connected'}
              </span>
              <span className="chip"><Clock3 size={14} /> {currentSessionLabel}</span>
              <span className="chip"><Pause size={14} /> Paused {currentPausedLabel}</span>
              <span className="chip"><BatteryFull size={14} /> {activeState?.batteryPercent ? `${activeState.batteryPercent}%` : 'Battery unknown'}</span>
            </div>
          </div>
          <div className="dashboardActions">
            {!hasKnownDevice && (
              <button className="primary wideAction" disabled={busy === 'scan'} onClick={scanDevices}>
                <Search size={17} /> Find TimeFlip
              </button>
            )}
            {hasKnownDevice && !selectedDeviceConnected && (
              <button className="primary wideAction" disabled={!selectedDevice || busy === 'connect'} onClick={() => runAction('connect', () => ConnectDevice(selectedDevice), 'Connect requested')}>
                <Plug size={17} /> Connect
              </button>
            )}
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
        )}

        {visiblePage === 'device' && (
        <section id="devices" className="grid two">
          <Panel title="Devices" icon={<Bluetooth size={18} />} description="Scan when setting up, then use this list to reconnect or switch devices later.">
            <div className="toolbar">
              <button disabled={busy === 'scan'} onClick={scanDevices}><Search size={16} /> Scan</button>
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
              {discovered.filter((device) => !state.devices?.some((known) => known.id === device.id)).map((device) => (
                <button key={device.id} className={`row discovery ${pairForm.deviceID === device.id ? 'selected' : ''}`} onClick={() => setPairForm((current) => ({ ...current, deviceID: device.id }))}>
                  <span>{device.name || 'Nearby TimeFlip'}</span>
                  <small>ready to pair</small>
                </button>
              ))}
              {state.devices?.length === 0 && <p className="empty">No known devices yet.</p>}
            </div>
          </Panel>

          <Panel title="Pairing" icon={<ShieldCheck size={18} />} description="Use the factory password for first pairing. Set a new password only when you want to change it on the device.">
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
                  New password, optional
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
        )}

        {visiblePage === 'config' && (
        <section className="grid two">
          <Panel title="Device details" icon={<Tag size={18} />} description="Rename the device for this app and check live hardware state before tuning.">
            <dl className="facts">
              <dt>Device</dt><dd>{selectedDeviceView?.displayName || selectedDevice || 'none'}</dd>
              <dt>Connection</dt><dd>{activeState?.connectionState || 'none'}</dd>
              <dt>Firmware</dt><dd>{selectedDeviceView?.firmwareVersion || 'unknown'}</dd>
              <dt>Battery</dt><dd>{activeState?.batteryPercent ? `${activeState.batteryPercent}%` : 'unknown'}</dd>
              <dt>Locked</dt><dd>{activeState?.locked ? 'yes' : 'no'}</dd>
              <dt>Paused</dt><dd>{activeState?.paused ? 'yes' : 'no'}</dd>
            </dl>
            <form className="form" onSubmit={saveDeviceName}>
              <label>
                Local device name
                <input maxLength={18} value={deviceNameForm} onChange={(event) => setDeviceNameForm(event.target.value)} placeholder="TimeFlip2" />
              </label>
              <button disabled={!selectedDevice || busy === 'deviceName'}><Save size={16} /> Save Name</button>
            </form>
	          </Panel>

	          <Panel title="Tap tuning" icon={<SlidersHorizontal size={18} />} description="Choose a feel, preview it on the connected device, then save when taps feel reliable.">
	            <form className="form" onSubmit={saveTapSettings}>
	              <dl className="facts compact">
	                <dt>Status</dt><dd>{selectedTapStatus}</dd>
	                <dt>Preview</dt><dd>{selectedTapTuning?.active ? 'active' : 'off'}</dd>
	                <dt>Detections</dt><dd>{selectedTapTuning?.detectedCount || 0}</dd>
	                <dt>Last tap</dt><dd>{selectedTapTuning?.lastObservation ? `Facet ${selectedTapTuning.lastObservation.facet} · ${formatTime(selectedTapTuning.lastObservation.occurredAt)}` : 'none'}</dd>
	              </dl>
	              <div className="presetGrid">
	                {tapPresets.map((preset) => (
	                  <button type="button" key={preset.id} disabled={!selectedDevice} onClick={() => applyTapPreset(preset)}>
	                    <SlidersHorizontal size={15} /> {preset.label}
	                  </button>
	                ))}
	              </div>
	              <div className="tapControlGrid">
	                <TapByteField label="Tap force threshold" unit="lower is more sensitive" value={tapForm.threshold} onChange={(value) => updateTapForm('threshold', value)} />
	                <TapByteField label="Repeat limit" unit="register ticks" value={tapForm.limit} onChange={(value) => updateTapForm('limit', value)} />
	                <TapByteField label="Debounce latency" unit="register ticks" value={tapForm.latency} onChange={(value) => updateTapForm('latency', value)} />
	                <TapByteField label="Detection window" unit="register ticks" value={tapForm.window} onChange={(value) => updateTapForm('window', value)} />
	              </div>
	              <div className="toolbar">
	                <button type="button" disabled={!selectedDevice || !selectedDeviceConnected || selectedTapTuning?.active || busy === 'tapTuningStart'} onClick={beginTapTuning}><Play size={16} /> Start preview</button>
	                <button type="button" disabled={!selectedDevice || !selectedTapTuning?.active || !selectedDeviceConnected || busy === 'tapTuningPreview'} onClick={previewTapTuning}><Check size={16} /> Apply temporary</button>
	                <button type="button" disabled={!selectedDevice || busy === 'tapReset'} onClick={resetTapForm}><RotateCcw size={16} /> Reset</button>
	                <button type="button" disabled={!selectedDevice || !selectedTapTuning?.active || busy === 'tapTuningCancel'} onClick={cancelTapTuning}><X size={16} /> Cancel</button>
	                <button className="primary" disabled={!selectedDevice || busy === 'tapSettings'}><Save size={16} /> Save Tap Feel</button>
	              </div>
	            </form>
	          </Panel>

	          <Panel title="LED feedback" icon={<Palette size={18} />} description="Set the device light brightness and blink length used for confirmations.">
	            <form className="form" onSubmit={saveLEDSettings}>
	              <dl className="facts compact">
	                <dt>Status</dt><dd>{selectedLEDSettings?.confirmedOnDevice ? 'confirmed on device' : selectedLEDSettings ? 'saved locally' : 'defaults'}</dd>
	              </dl>
	              <div className="tapControlGrid">
	                <RangeNumberField label="Brightness" min={1} max={100} unit="percent" value={ledForm.brightnessPercent} onChange={(value) => updateLEDForm('brightnessPercent', value)} />
	                <RangeNumberField label="Blink length" min={5} max={60} unit="seconds" value={ledForm.blinkSeconds} onChange={(value) => updateLEDForm('blinkSeconds', value)} />
	              </div>
              <button className="primary" disabled={!selectedDevice || busy === 'ledSettings'}><Save size={16} /> Save LED Feedback</button>
	            </form>
	          </Panel>
        </section>
        )}

        {visiblePage === 'device' && hasPairedDevice && (
          <section className="grid two">
            <Panel title="Advanced device controls" icon={<KeyRound size={18} />} description="Password changes, OS unpairing, and factory reset are rare maintenance actions.">
              <details className="advancedPanel">
                <summary><KeyRound size={16} /> Change password</summary>
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
              </details>
              <details className="advancedPanel dangerZone">
                <summary><Trash2 size={16} /> Unpair or reset</summary>
                <form className="form" onSubmit={unpairDevice}>
                  <p className="helper">Unpairing removes the local relationship with this app. Factory reset also asks the device to clear its own pairing state.</p>
                  <div className="formGrid">
                    <label className="check"><input type="checkbox" checked={unpairForm.factoryReset} onChange={(event) => setUnpairForm({ ...unpairForm, factoryReset: event.target.checked })} /> Factory reset</label>
                    <label className="check"><input type="checkbox" checked={unpairForm.allowOSUnpairing} onChange={(event) => setUnpairForm({ ...unpairForm, allowOSUnpairing: event.target.checked })} /> Remove from OS Bluetooth list</label>
                  </div>
                  <button className="danger" disabled={!selectedDevice || busy === 'unpair'}><Trash2 size={16} /> Unpair</button>
                </form>
              </details>
            </Panel>
          </section>
        )}

        {visiblePage === 'device' && workflow && <WorkflowStatus workflow={workflow} />}

        {visiblePage === 'facets' && (
        <section id="facets" className="band">
          <div className="sectionTitle">
            <h2><SlidersHorizontal size={20} /> Facets</h2>
            <div className="sectionActions">
              <p>Give each physical side a task, colour, and icon so routine tracking is glanceable.</p>
              <button type="button" className="danger" disabled={!selectedDevice || busy === 'resetFacets'} onClick={resetAllFacets}>
                <RotateCcw size={16} /> Reset All Facets
              </button>
            </div>
          </div>
          <div className="facetLayout">
            <div className="facetGrid">
              {Array.from({ length: 12 }, (_, index) => {
                const facet = index + 1;
                const cfg = state.facetConfigs?.find((item) => item.deviceID === selectedDevice && item.facet === facet);
                const isLiveFacet = liveFacet === facet;
                return (
                  <button className={`facet ${selectedFacet === facet ? 'selected' : ''} ${isLiveFacet ? 'current' : ''}`} key={facet} onClick={() => setSelectedFacet(facet)}>
                    <span className="swatch" style={{ background: cfg?.color || '#d8dee9' }} />
                    <strong><IconGlyph name={cfg?.icon} size={16} /> <span>{cfg?.label || `Facet ${facet}`}</span></strong>
                    <small>{facetTileKind(cfg)}{isLiveFacet ? <span className="currentFacetBadge">Current</span> : null}</small>
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
                  <strong><IconGlyph name={selectedFacetConfig?.icon} size={16} /> <span>{selectedFacetSavedLabel}</span></strong>
                  <small>{selectedFacetSavedKind}{selectedFacetConfig?.assignedOnDevice ? ' · confirmed on device' : ''}</small>
                </div>
              </div>
              <div className="fieldGroup">
                <span className="fieldTitle">Facet type</span>
                <div className="segmented" role="group" aria-label="Facet type">
                  <button type="button" className={taskForm.mode === 'task' ? 'selected' : ''} onClick={() => chooseFacetMode('task')}><BriefcaseBusiness size={16} /> Task</button>
                  <button type="button" className={taskForm.mode === 'pomodoro' ? 'selected' : ''} onClick={() => chooseFacetMode('pomodoro')}><Clock3 size={16} /> Pomodoro</button>
                  <button type="button" className={taskForm.mode === 'pause' ? 'selected' : ''} onClick={() => chooseFacetMode('pause')}><Pause size={16} /> Pause</button>
                </div>
              </div>
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
              </div>
              <div className="fieldGroup">
                <span className="fieldTitle">Icon</span>
                <IconDropdown
                  options={stickerIconOptions}
                  value={taskFormUsesCustomIcon ? 'custom' : taskForm.icon}
                  customOption={taskFormUsesCustomIcon ? { value: 'custom', label: 'Custom', icon: resolveIconComponent(taskForm.icon) } : { value: 'custom', label: 'Custom', icon: Palette }}
                  onChange={(value) => updateFacetForm({ icon: value === 'custom' ? customIconOptions[0]?.value || defaultTask.icon : value })}
                />
              </div>
              {taskFormUsesCustomIcon && (
                <div className="fieldGroup">
                  <span className="fieldTitle">Custom icon</span>
                  <IconDropdown options={customIconOptions} value={taskForm.icon} onChange={(value) => updateFacetForm({ icon: value })} />
                </div>
              )}
              <div className="fieldGroup">
                <span className="fieldTitle">Colour</span>
                <div className="colorPickerRow">
                  <input type="color" value={taskForm.color} onChange={(event) => updateFacetForm({ color: event.target.value })} aria-label="Facet colour" />
                  <div className="colorChoices" aria-label="Suggested facet colours">
                    {facetColorOptions.map((color) => (
                      <button
                        type="button"
                        className={`colorChoice ${normaliseColor(taskForm.color, '') === color ? 'selected' : ''}`}
                        key={color}
                        onClick={() => updateFacetForm({ color })}
                        style={{ background: color }}
                        title={color}
                      />
                    ))}
                  </div>
                </div>
              </div>
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
                <button
                  type="button"
                  className="danger"
                  disabled={!selectedDevice || busy === 'clearFacet'}
                  onClick={clearFacet}
                >
                  <Trash2 size={16} /> {clearFacetArmed ? `Confirm Clear Facet ${selectedFacet}` : 'Clear Facet'}
                </button>
                <button type="button" disabled={taskForm.mode === 'pause' || busy === 'task'} onClick={saveTask}><Plus size={16} /> Save Task</button>
                <button className="primary" disabled={!selectedDevice || busy === 'facet'}><Save size={16} /> Save Facet {selectedFacet}</button>
              </div>
            </form>
          </div>
        </section>
        )}

        {visiblePage === 'track' && (
        <section id="history" className="band">
          <div className="sectionTitle">
            <h2><History size={20} /> Task Sessions</h2>
            <p>Recent local sessions, with paused time separated from elapsed time.</p>
          </div>
          <div className="sessionList">
            {state.sessions?.slice(0, 12).map((session) => (
              <div className="session" key={session.id}>
                <IconBadge name={session.taskIconSnapshot} color={session.taskColorSnapshot || '#d8dee9'} />
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
        )}

        {visiblePage === 'config' && (
        <section id="settings" className="band">
          <div className="sectionTitle">
            <h2><Settings size={20} /> Settings</h2>
            <p>Advanced reconnect policy and local storage details.</p>
          </div>
          <p className="storagePath">{state.config?.databasePath || 'Local app config'}</p>
          <details className="advancedPanel" open>
            <summary><Settings size={16} /> Reconnect timing</summary>
            <form className="settingsGrid" onSubmit={saveSettings}>
              <NumberField label="Communication timeout" unit="seconds" value={settingsForm.communicationTimeoutSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, communicationTimeoutSeconds: value })} />
              <NumberField label="Command timeout" unit="seconds" value={settingsForm.commandTimeoutSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, commandTimeoutSeconds: value })} />
              <NumberField label="Initial retry" unit="seconds" value={settingsForm.initialRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, initialRetrySeconds: value })} />
              <NumberField label="Medium retry" unit="seconds" value={settingsForm.mediumRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, mediumRetrySeconds: value })} />
              <NumberField label="Long retry" unit="seconds" value={settingsForm.longRetrySeconds} onChange={(value) => setSettingsForm({ ...settingsForm, longRetrySeconds: value })} />
              <NumberField label="Mark offline after" unit="seconds" value={settingsForm.offlineAfterSeconds} onChange={(value) => setSettingsForm({ ...settingsForm, offlineAfterSeconds: value })} />
              <NumberField label="Failure threshold" unit="failures" value={settingsForm.offlineAfterFailures} onChange={(value) => setSettingsForm({ ...settingsForm, offlineAfterFailures: value })} />
              <button className="primary" disabled={busy === 'settings'}><Save size={16} /> Save Settings</button>
            </form>
          </details>
        </section>
        )}
      </section>
    </main>
  );
}

function Panel({ title, icon, description = '', children }) {
  return (
    <section className="panel">
      <h2>{icon}{title}</h2>
      {description ? <p className="panelIntro">{description}</p> : null}
      {children}
    </section>
  );
}

function SandtimerLogo() {
  return (
    <svg viewBox="0 0 64 64" focusable="false">
      <rect x="5" y="5" width="54" height="54" rx="14" />
      <path className="sand" d="M23 22h18L32 32zM24 44h16l-8-10z" />
      <path d="M21 18h22M21 46h22M22 19l10 13M42 19L32 32M32 32L22 45M32 32l10 13" />
    </svg>
  );
}

function DismissibleNotice({ kind, onDismiss, children }) {
  return (
    <div className={`notice ${kind}`}>
      <span>{children}</span>
      <button type="button" className="noticeDismiss" onClick={onDismiss} aria-label="Dismiss message">
        <X size={16} />
      </button>
    </div>
  );
}

function NavButton({ page, currentPage, onNavigate, icon, children }) {
  return (
    <button type="button" className={`navButton ${currentPage === page ? 'selected' : ''}`} onClick={() => onNavigate(page)}>
      {icon}{children}
    </button>
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

function TapByteField({ label, unit, value, onChange }) {
  return (
    <label className="tapControl">
      <span className="fieldLabel">
        <span>{label}</span>
        <small>{unit}</small>
      </span>
      <div className="rangeRow">
        <input type="range" min={0} max={255} value={value} aria-label={`${label}, ${unit}`} onChange={(event) => onChange(Number(event.target.value))} />
        <input type="number" min={0} max={255} value={value} aria-label={`${label} byte value`} onChange={(event) => onChange(Number(event.target.value))} />
      </div>
    </label>
  );
}

function RangeNumberField({ label, unit, value, onChange, min = 0, max = 255 }) {
  return (
    <label className="tapControl">
      <span className="fieldLabel">
        <span>{label}</span>
        <small>{unit}</small>
      </span>
      <div className="rangeRow">
        <input type="range" min={min} max={max} value={value} aria-label={`${label}, ${unit}`} onChange={(event) => onChange(Number(event.target.value))} />
        <input type="number" min={min} max={max} value={value} aria-label={`${label} value`} onChange={(event) => onChange(Number(event.target.value))} />
      </div>
    </label>
  );
}

function IconGlyph({ name = defaultTask.icon, size = 16 }) {
  const Glyph = resolveIconComponent(name);
  return <Glyph size={size} aria-hidden="true" />;
}

function IconChoiceButton({ option, selected, onClick }) {
  const Glyph = option.icon;
  return (
    <button type="button" className={`iconChoice ${selected ? 'selected' : ''}`} onClick={onClick}>
      <Glyph size={18} aria-hidden="true" />
      <span>{option.label}</span>
    </button>
  );
}

function IconDropdown({ options, value, onChange, customOption = null }) {
  const [open, setOpen] = useState(false);
  const allOptions = customOption ? [...options, customOption] : options;
  const selected = allOptions.find((option) => option.value === value) || allOptions[0];
  const SelectedGlyph = selected?.icon || Tag;
  return (
    <div className="iconDropdown">
      <button type="button" className="iconDropdownButton" onClick={() => setOpen((current) => !current)} aria-expanded={open}>
        <SelectedGlyph size={18} aria-hidden="true" />
        <span>{selected?.label || 'Choose icon'}</span>
        <ChevronDown size={16} aria-hidden="true" />
      </button>
      {open && (
        <div className="iconDropdownMenu" role="listbox">
          {allOptions.map((option) => (
            <IconChoiceButton
              key={option.value}
              option={option}
              selected={value === option.value}
              onClick={() => {
                onChange(option.value);
                setOpen(false);
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function IconBadge({ name = 'tag', color = '#d8dee9', size = 18 }) {
  return (
    <span className="iconBadge" style={{ background: normaliseColor(color, '#d8dee9') }}>
      <IconGlyph name={name} size={size} />
    </span>
  );
}

function isStickerIcon(name) {
  return stickerIconOptions.some((option) => option.value === name);
}

function isCustomTaskIcon(name) {
  return Boolean(name) && !isStickerIcon(name) && customIconOptions.some((option) => option.value === name);
}

function resolveIconComponent(name) {
  const stickerOption = stickerIconOptions.find((option) => option.value === name);
  if (stickerOption?.icon) {
    return stickerOption.icon;
  }
  const customOption = customIconOptions.find((option) => option.value === name);
  if (customOption?.icon) {
    return customOption.icon;
  }
  return legacyIconAliases[name] || Tag;
}

function humaniseIconName(name) {
  return String(name || '')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function UXIcon({ size = 16 }) {
  return <span className="letterIcon" style={{ fontSize: Math.max(10, Math.round(size * 0.72)) }}>UX</span>;
}

function QuotationIcon({ size = 16 }) {
  return <span className="mathIcon" style={{ fontSize: Math.max(9, Math.round(size * 0.58)) }}>+−×÷</span>;
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
      icon: config.icon || defaultTask.icon,
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
  return iconDisplayLabel(config.icon);
}

function iconDisplayLabel(name) {
  const stickerOption = stickerIconOptions.find((option) => option.value === name);
  if (stickerOption) {
    return stickerOption.label;
  }
  return name || 'Task';
}

function selectedFacetLabel(config, facet) {
  if (!config?.taskID && !config?.isPauseAssignment && !config?.label) {
    return `Facet ${facet} is unassigned`;
  }
  return config.label || `Facet ${facet}`;
}

function isPairedDevice(device) {
  return device?.pairingState === 'paired' || Boolean(device?.hasPassword);
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

function mergeTapTuningState(current, tapTuningState) {
  const tapTuningStates = current.tapTuningStates || [];
  const nextStates = tapTuningState.active
    ? upsertByDevice(tapTuningStates, tapTuningState)
    : tapTuningStates.filter((state) => state.deviceID !== tapTuningState.deviceID);
  return { ...current, tapTuningStates: nextStates };
}

function mergeTapTuningObservation(current, observation) {
  const tapTuningStates = current.tapTuningStates || [];
  const found = tapTuningStates.some((state) => state.deviceID === observation.deviceID);
  const nextObservation = { lastObservation: observation };
  return {
    ...current,
    tapTuningStates: found
      ? tapTuningStates.map((state) => (state.deviceID === observation.deviceID
        ? { ...state, ...nextObservation, detectedCount: Number(state.detectedCount || 0) + 1 }
        : state))
      : tapTuningStates,
  };
}

function upsertByDevice(items = [], item) {
  const found = items.some((existing) => existing.deviceID === item.deviceID);
  return found
    ? items.map((existing) => (existing.deviceID === item.deviceID ? { ...existing, ...item } : existing))
    : [...items, item];
}

function mergeAppState(current, next) {
  return {
    ...next,
    states: mergeDeviceStates(current.states, next.states),
  };
}

function replaceFacetConfigsForDevice(state, deviceID, configs = []) {
  const remaining = state.facetConfigs?.filter((item) => item.deviceID !== deviceID) || [];
  return {
    ...state,
    facetConfigs: [...remaining, ...configs.map((item) => ({ ...item, deviceID }))],
  };
}

function upsertFacetConfig(state, config) {
  const facetConfigs = state.facetConfigs || [];
  const found = facetConfigs.some((item) => item.deviceID === config.deviceID && item.facet === config.facet);
  return {
    ...state,
    facetConfigs: found
      ? facetConfigs.map((item) => (item.deviceID === config.deviceID && item.facet === config.facet ? { ...item, ...config } : item))
      : [...facetConfigs, config],
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

function formatTime(value) {
  if (!value) {
    return 'not recorded';
  }
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) {
    return 'not recorded';
  }
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
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
