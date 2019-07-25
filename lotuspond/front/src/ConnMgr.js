import React from 'react';
import Cristal from 'react-cristal'

async function awaitReducer(prev, c) {
  await prev
  await c
}

class ConnMgr extends React.Component {
  constructor(props) {
    super(props)

    this.connect = this.connect.bind(this)
    this.connectAll = this.connectAll.bind(this)
    this.connect1 = this.connect1.bind(this)
    this.connectChain = this.connectChain.bind(this)
    this.getActualState = this.getActualState.bind(this)

    this.state = {conns: {}, lock: true}

    this.getActualState()
    setInterval(this.getActualState, 2000)
  }

  async getActualState() {
    const nodes = this.props.nodes
    let keys = Object.keys(nodes)

    await keys.filter((_, i) => i > 0).map(async (kfrom, i) => {
      await keys.filter((_, j) => i >= j).map(async kto => {

        const fromNd = this.props.nodes[kfrom]
        const toNd = this.props.nodes[kto]

        const connectedness = await fromNd.conn.call('Filecoin.NetConnectedness', [toNd.peerid])

        this.setState(prev => ({conns: {...prev.conns, [`${kfrom},${kto}`]: connectedness === 1}}))
      }).reduce(awaitReducer)
    }).reduce(awaitReducer)

    this.setState({lock: false})
  }

  async connect(action, from, to) {
    const fromNd = this.props.nodes[from]
    const toNd = this.props.nodes[to]

    if (action) {
      const toPeerInfo = await toNd.conn.call('Filecoin.NetAddrsListen', [])

      await fromNd.conn.call('Filecoin.NetConnect', [toPeerInfo])
    } else {
      await fromNd.conn.call('Filecoin.NetDisconnect', [toNd.peerid])
    }

    this.setState(prev => ({conns: {...prev.conns, [`${from},${to}`]: action}}))
  }

  connectAll(discon) {
    return () => {
      const nodes = this.props.nodes
      let keys = Object.keys(nodes)

      keys.filter((_, i) => i > 0).forEach((kfrom, i) => {
        keys.filter((_, j) => i >= j).forEach((kto, i) => {
          this.connect(!discon, kfrom, kto)
        })
      })
    }
  }

  connect1() {
    const nodes = this.props.nodes
    let keys = Object.keys(nodes)

    keys.filter((_, i) => i > 0).forEach((k, i) => {
      this.connect(true, k, keys[0])
    })
  }

  connectChain() {
    const nodes = this.props.nodes
    let keys = Object.keys(nodes)

    keys.filter((_, i) => i > 0).forEach((k, i) => {
      this.connect(true, k, keys[i])
    })
  }

  render() {
    const nodes = this.props.nodes
    let keys = Object.keys(nodes)

    const rows = keys.filter((_, i) => i > 0).map((k, i) => {
      const cols = keys.filter((_, j) => i >= j).map((kt, i) => {
        const checked = this.state.conns[`${k},${kt}`] === true

        return (
          <td key={k + "," + kt}>
            <input checked={checked} disabled={this.state.lock} type="checkbox" onChange={e => this.connect(e.target.checked, k, kt)}/>
          </td>
        )
      })
      return (
          <tr key={k}><td>{k}</td>{cols}</tr>
      )
    })

    return(
      <Cristal title={`Connection Manager${this.state.lock ? ' (syncing)' : ''}`}>
        <table>
          <thead><tr><td></td>{keys.slice(0, -1).map((i) => (<td key={i}>{i}</td>))}</tr></thead>
          <tbody>{rows}</tbody>
        </table>
        <div>
          <button onClick={this.connectAll(true)}>DisonnAll</button>
          <button onClick={this.connectAll(false)}>ConnAll</button>
          <button onClick={this.connect1}>Conn1</button>
          <button onClick={this.connectChain}>ConnChain</button>
        </div>
      </Cristal>
    )
  }
}

export default ConnMgr