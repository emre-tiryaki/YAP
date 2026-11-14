// Settings stored in the database, so they follow the user to every device.
export type UserSettings = {
  autosaveEnabled: boolean;
  autosaveDelayMs: number;
};

export type UpdateUserSettingsInput = Partial<UserSettings>;
