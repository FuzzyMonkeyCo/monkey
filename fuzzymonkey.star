# Invariants of our APIs expressed in a Python-like language

mode = Env('TESTING_WHAT', 'jsonplaceholder')
host, spec = 'https://jsonplaceholder.typicode.com/', 'lib/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml'
if mode == 'jsonplaceholder':
    pass
elif mode == 'other-thing':
    spec = 'some/other/spec.json'
else:
    fail("Unhandled testing mode '{}'".format(mode))
print('Now testing {}.'.format(spec))

OpenAPIv3(
    file = spec,

    host = host,
    # authorization = 'Bearer ' + ...,

    ExecReset = '''
    printf 'Resetting state...\n'
    '''
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
    probe = ('monkey', 'http', 'response'),
    predicate = lambda response: all([
        response['request']['method'] == 'GET',
        response['request']['path'] == '/csgo/weapons',
        response['status_code'] == 200,
    ]),
    # predicate = None,
    # match = {
    #     'request': {'method': 'GET', 'path': '/csgo/weapons'},
    #     'status_code': 200,
    # },
    action = actionAfterWeapons,
)

TriggerActionAfterProbe(
    probe = ('http', 'response'),
    predicate = lambda response: all([
        response['request']['method'] == 'GET',
        response['request']['route'] == '/csgo/weapons/:weapon_id',
        response['status_code'] in range(200, 299),
        response['body']['id'] in State['weapons'],
    ]),
    # match = None,
    action = actionAfterGetExistingWeapon,
)
