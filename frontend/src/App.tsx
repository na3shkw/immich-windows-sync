import { useState, useEffect } from 'react';
import { LoadConfig, SaveConfig, SelectFolder } from '../wailsjs/go/main/App';
import { config } from '../wailsjs/go/models';

type Page = 'connection' | 'folders' | 'status';
type SaveStatus = 'idle' | 'saved' | 'error';

function App() {
  const [activePage, setActivePage] = useState<Page>('connection');
  const [cfg, setCfg] = useState<config.Config>(
    config.Config.createFrom({ immich: { serverURL: '', apiKey: '' }, targetFolders: [] })
  );
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle');

  useEffect(() => {
    LoadConfig().then(setCfg);
  }, []);

  const handleSave = async () => {
    try {
      await SaveConfig(cfg);
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 2000);
    } catch {
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 2000);
    }
  };

  const handleAddFolder = async () => {
    const folder = await SelectFolder();
    if (!folder) return;
    if (cfg.targetFolders?.includes(folder)) return;
    const updated = config.Config.createFrom({
      ...cfg,
      targetFolders: [...(cfg.targetFolders ?? []), folder],
    });
    setCfg(updated);
    await SaveConfig(updated);
  };

  const handleRemoveFolder = async (folder: string) => {
    const updated = config.Config.createFrom({
      ...cfg,
      targetFolders: cfg.targetFolders?.filter((f) => f !== folder) ?? [],
    });
    setCfg(updated);
    await SaveConfig(updated);
  };

  const navItems: { id: Page; label: string; icon: string }[] = [
    { id: 'connection', label: 'Connection Settings', icon: 'settings_input_component' },
    { id: 'folders', label: 'Folder Management', icon: 'folder_shared' },
    { id: 'status', label: 'Sync Status', icon: 'sync_saved_locally' },
  ];

  return (
    <div className="flex h-screen w-screen bg-surface text-on-surface font-body overflow-hidden">
      {/* Sidebar */}
      <nav className="w-64 h-full bg-surface-container-low flex flex-col py-8 px-4 shrink-0">
        <div className="mb-10 px-2">
          <h1 className="text-base font-bold text-primary uppercase tracking-wide">immich-sync</h1>
          <p className="text-[10px] text-on-surface-variant uppercase tracking-widest mt-1 opacity-60">Windows Sync</p>
        </div>

        <div className="flex-1 space-y-1">
          {navItems.map((item) => (
            <button
              key={item.id}
              onClick={() => setActivePage(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-all duration-200 ${
                activePage === item.id
                  ? 'bg-surface-container text-primary border-l-2 border-primary font-semibold'
                  : 'text-on-surface-variant hover:text-primary hover:bg-surface-container'
              }`}
            >
              <span className="material-symbols-outlined text-[20px]">{item.icon}</span>
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </nav>

      {/* Main Content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="h-16 flex items-center px-10 bg-surface border-b border-outline-variant/10 shrink-0">
          <h2 className="text-xl font-bold tracking-tight text-primary">
            {navItems.find((i) => i.id === activePage)?.label}
          </h2>
        </header>

        {/* Page Content */}
        <div className="flex-1 overflow-y-auto p-10">
          {activePage === 'connection' && (
            <ConnectionPage cfg={cfg} setCfg={setCfg} onSave={handleSave} saveStatus={saveStatus} />
          )}
          {activePage === 'folders' && (
            <FolderManagementPage
              folders={cfg.targetFolders ?? []}
              onAdd={handleAddFolder}
              onRemove={handleRemoveFolder}
            />
          )}
          {activePage === 'status' && (
            <div className="text-on-surface-variant">Coming soon...</div>
          )}
        </div>
      </main>
    </div>
  );
}

function ConnectionPage({
  cfg,
  setCfg,
  onSave,
  saveStatus,
}: {
  cfg: config.Config;
  setCfg: (cfg: config.Config) => void;
  onSave: () => void;
  saveStatus: SaveStatus;
}) {
  const [showApiKey, setShowApiKey] = useState(false);

  const updateImmich = (field: keyof config.ImmichConfig, value: string) => {
    setCfg(
      config.Config.createFrom({
        ...cfg,
        immich: { ...cfg.immich, [field]: value },
      })
    );
  };

  return (
    <div className="max-w-xl">
      <p className="text-on-surface-variant text-sm mb-8">
        Configure your Immich server connection.
      </p>

      <div className="bg-surface-container rounded-xl p-8 space-y-6">
        {/* Server URL */}
        <div className="space-y-2">
          <label className="text-[10px] uppercase tracking-widest font-bold text-primary opacity-80">
            Server URL
          </label>
          <div className="relative">
            <input
              type="text"
              value={cfg.immich?.serverURL ?? ''}
              onChange={(e) => updateImmich('serverURL', e.target.value)}
              placeholder="http://192.168.1.x:2283"
              className="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3 text-on-surface text-sm font-mono focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 transition-all"
            />
            <span className="absolute right-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-on-surface-variant/40 text-[18px]">
              link
            </span>
          </div>
        </div>

        {/* API Key */}
        <div className="space-y-2">
          <label className="text-[10px] uppercase tracking-widest font-bold text-primary opacity-80">
            API Key
          </label>
          <div className="relative">
            <input
              type={showApiKey ? 'text' : 'password'}
              value={cfg.immich?.apiKey ?? ''}
              onChange={(e) => updateImmich('apiKey', e.target.value)}
              placeholder="Your Immich API key"
              className="w-full bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3 text-on-surface text-sm font-mono focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 transition-all"
            />
            <button
              type="button"
              onClick={() => setShowApiKey((v) => !v)}
              className="absolute right-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-on-surface-variant hover:text-primary transition-colors text-[18px]"
            >
              {showApiKey ? 'visibility_off' : 'visibility'}
            </button>
          </div>
        </div>

        {/* Save Button */}
        <div className="flex items-center justify-end gap-4 pt-2">
          {saveStatus === 'saved' && (
            <span className="text-xs text-secondary flex items-center gap-1">
              <span className="material-symbols-outlined text-[16px]">check_circle</span>
              Saved
            </span>
          )}
          {saveStatus === 'error' && (
            <span className="text-xs text-error flex items-center gap-1">
              <span className="material-symbols-outlined text-[16px]">error</span>
              Failed to save
            </span>
          )}
          <button
            onClick={onSave}
            className="bg-gradient-to-br from-primary to-primary-container text-on-primary-fixed px-8 py-2.5 rounded-lg font-bold text-sm shadow-lg shadow-primary/10 hover:shadow-primary/20 active:scale-95 transition-all"
          >
            Save Changes
          </button>
        </div>
      </div>
    </div>
  );
}

function FolderManagementPage({
  folders,
  onAdd,
  onRemove,
}: {
  folders: string[];
  onAdd: () => void;
  onRemove: (folder: string) => void;
}) {
  return (
    <div className="max-w-xl">
      <p className="text-on-surface-variant text-sm mb-8">
        Select folders to monitor and sync to Immich.
      </p>

      <div className="bg-surface-container rounded-xl p-8 space-y-6">
        {/* Folder List */}
        <div className="space-y-2">
          {folders.length === 0 ? (
            <p className="text-on-surface-variant text-sm text-center py-6">
              No folders added yet.
            </p>
          ) : (
            folders.map((folder) => (
              <div
                key={folder}
                className="flex items-center justify-between bg-surface-container-lowest border border-outline-variant/20 rounded-lg px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <span className="material-symbols-outlined text-primary text-[20px] shrink-0">folder</span>
                  <span className="text-sm font-mono text-on-surface break-all">{folder}</span>
                </div>
                <button
                  onClick={() => onRemove(folder)}
                  className="material-symbols-outlined text-on-surface-variant hover:text-error transition-colors text-[20px] shrink-0 ml-4"
                >
                  delete
                </button>
              </div>
            ))
          )}
        </div>

        {/* Add Button */}
        <button
          onClick={onAdd}
          className="w-full flex items-center justify-center gap-2 border border-dashed border-outline-variant/40 hover:border-primary text-on-surface-variant hover:text-primary rounded-lg py-3 text-sm font-medium transition-all"
        >
          <span className="material-symbols-outlined text-[20px]">add</span>
          Add Folder
        </button>
      </div>
    </div>
  );
}

export default App;
