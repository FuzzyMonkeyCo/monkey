# Invariants of our APIs expressed in a Python-like language

print('THIS_ENVIRONMENT_VARIABLE is', Env('THIS_ENVIRONMENT_VARIABLE','unset'))

host, spec = 'https://jsonplaceholder.typicode.com/', None
mode = Env('TESTING_WHAT')
if mode == '':
    spec = 'lib/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml'
elif mode == 'other-thing':
    pass
else:
    fail("Unhandled testing mode '{}'".format(mode))
print('Now testing {}.'.format(spec))

OpenAPIv3(
    file = spec,

    host = host,
    # header_authorization = 'Bearer ' + ...,

    ExecReset = '''
    printf 'Resetting state...\n'
    '''
)


# State is optional but HAS TO be a Dict.
State = {
    'weapons': {},
}

def actionAfterWeapons(State, response):
    print('### State =', State)
    print("!!! actionAfterWeapons", response)
    State['bla'] = 42
    print('### State =', State)
    # Response has already been validated and JSON decoded
    body = response['body']
    print("Setting thing {}".format(body['id']))
    # Set some state
    State['weapons'][ body['id'] ] = body
    print('### State =', State)
    return State

def actionAfterGetExistingWeapon(State, response):
    print('!!! actionAfterGetWeapon', response)
    weapon_id = int(response['request']['url'][-1])
    body = response['body']
    # Ensure an API contract
    #assert.eq(weapon_id, body['id'])
    # Implied: if weapon_id in StateGet('weapons'):
    # Verify state
    #AssertThat(body).equals(weapons[weapon_id])
    if body != State['weapons'][weapon_id]:
        fail("wrong data for weapon:", weapon_id,
             "expected", State['weapons'][weapon_id],
             "got", body)
    return State

TriggerActionAfterProbe(
    name = 'Collect things',
    probe = ('monkey', 'http', 'response'),
    predicate = lambda State, response: all([
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
    name = 'Ensure things match collected',
    probe = ('http', 'response'),
    predicate = lambda State, response: all([
        response['request']['method'] == 'GET',
        response['request']['route'] == '/csgo/weapons/:weapon_id',
        response['status_code'] in range(200, 299),
        response['body']['id'] in State['weapons'],
    ]),
    # match = None,
    action = actionAfterGetExistingWeapon,
)
