spec = 'dev-spec.json'
if Env('SUT_IS_PROD') != '':
    spec = 'normal_spec.yml'
print('Now testing {}.'.format(spec))

bearerAuth = Env("SOME_AUTH_TOKEN") or fail("unset SOME_AUTH_TOKEN")
bearerAuth = 'Bearer ' + bearerAuth
backendHost = Env("SOME_HOST_URL") or fail("unset SOME_HOST_URL")

OpenAPIv3(
    file = 'openapi/{}'.format(spec),

    host = backendHost,
    authorization = bearerAuth,

    exec_reset = 'printf "Resetting state...\n"'
)


State = {
    'weapons': {},
}

def actionAfterWeapons(response):
    print(StateGet())
    print("!!! actionAfterWeapons", response)
    return
    # Response has already been validated and JSON decoded
    body = response['body']['as_json']
    # Set some state
    weapons = StateGet('weapons')
    weapons[ body['id'] ] = body
    StateUpdate('weapons', weapons)

def actionAfterGetExistingWeapon(response):
    print('actionAfterGetWeapon', response)
    weapon_id = int(response['request']['url'][-1])
    body = response['body']['as_json']
    # Ensure an API contract
    #assert.eq(weapon_id, body['id'])
    # Implied: if weapon_id in StateGet('weapons'):
    # Verify state
    #AssertThat(body).equals(weapons[weapon_id])
    if body != StateGet('weapons')[weapon_id]:
        fail("wrong data for weapon:", weapon_id,
             "expected", StateGet('weapons')[weapon_id],
             "got", body)

# There MUST NOT be any upper case exports.
# State = {"strk": v0} # : State is optional but HAS TO be a Dict.
# StateSet(k, v)
# StateUpdate(k, v2)
# StateGet(k)
# StateItems()
# StateKeys()
# StateDelete(k)

TriggerActionAfterProbe(
    probe = 'monkey:http:response',
    predicate = None,
    match = {
        'request': {'method': 'GET', 'path': '/csgo/weapons'},
        'status_code': 200,
    },
    action = actionAfterWeapons,
)

TriggerActionAfterProbe(
    probe = 'monkey:http:response',
    predicate = lambda response: all([
        response['request']['method'] == 'GET',
        response['request']['route'] == '/csgo/weapons/:weapon_id',
        response['status_code'] in range(200, 299),
        response['body']['id'] in State['weapons'],
    ]),
    match = None,
    action = actionAfterGetExistingWeapon,
)
