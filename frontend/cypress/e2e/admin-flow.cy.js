describe('MDM Matrix admin workflow', () => {
  it('restores a locked device and offers Unlock after reload', () => {
    localStorage.setItem('mdm_token', 'test-token');
    cy.intercept('GET', '**/devices', { body: [{ id: 'fleet-device-001', name: 'Locked simulator', status: 'online', locked: true, battery: 85, last_seen: new Date().toISOString() }] }).as('devices');
    cy.intercept('GET', '**/commands?limit=*', { body: [] });
    cy.intercept('GET', '**/metrics/commands', { body: { total: 0, active: 0, completed: 0, failed: 0, average_completion_ms: 0 } });
    cy.intercept('POST', '**/devices/fleet-device-001/command', { statusCode: 202, body: { command_id: 'unlock-1' } }).as('unlock');
    cy.visit('/');
    cy.wait('@devices');
    cy.contains('button', 'Unlock').click();
    cy.wait('@unlock').its('request.body').should('deep.equal', { type: 'unlock' });
    cy.contains('Unlock sent').should('be.visible');
  });

  it('logs in, loads the fleet, and dispatches a command', () => {
    cy.intercept('POST', '**/login', { statusCode: 200, body: { token: 'test-token' } }).as('login');
    cy.intercept('GET', '**/devices', {
      statusCode: 200,
      body: [{ id: 'fleet-device-001', name: 'Simulated Device 1', status: 'online', battery: 85, last_seen: new Date().toISOString() }],
    }).as('devices');
    cy.intercept('GET', '**/commands?limit=*', { statusCode: 200, body: [] }).as('commands');
    cy.intercept('GET', '**/metrics/commands', { statusCode: 200, body: { total: 0, active: 0, completed: 0, failed: 0, average_completion_ms: 0 } }).as('metrics');
    cy.intercept('POST', '**/devices/fleet-device-001/command', {
      statusCode: 202,
      body: { command_id: 'command-1', status: 'dispatched_or_queued' },
    }).as('dispatch');

    cy.visit('/');
    cy.get('input[autocomplete="username"]').type('admin');
    cy.get('input[autocomplete="current-password"]').type('correct-password');
    cy.contains('button', 'Sign in securely').click();
    cy.wait('@login');
    cy.wait('@devices');
    cy.contains('Simulated Device 1').should('be.visible');
    cy.contains('button', 'Lock').click();
    cy.wait('@dispatch').its('request.body').should('deep.equal', { type: 'lock' });
    cy.contains('Lock sent').should('be.visible');
  });
});
